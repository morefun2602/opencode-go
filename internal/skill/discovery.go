package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type IndexSkill struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

type Index struct {
	Skills []IndexSkill `json:"skills"`
}

type Discovery struct {
	CacheDir string
	Log      *slog.Logger
	Client   *http.Client
}

func NewDiscovery(cacheDir string, log *slog.Logger) *Discovery {
	return &Discovery{
		CacheDir: cacheDir,
		Log:      log,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Pull fetches a remote skill index and downloads skill files to the local cache.
// Returns a list of local cache directories containing successfully downloaded skills.
func (d *Discovery) Pull(rawURL string) []string {
	base := rawURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	indexURL := base + "index.json"

	d.Log.Info("fetching skill index", "url", indexURL)

	resp, err := d.Client.Get(indexURL)
	if err != nil {
		d.Log.Error("failed to fetch skill index", "url", indexURL, "err", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		d.Log.Error("skill index returned non-OK status", "url", indexURL, "status", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		d.Log.Error("failed to read skill index", "url", indexURL, "err", err)
		return nil
	}

	var idx Index
	if err := json.Unmarshal(body, &idx); err != nil {
		d.Log.Error("failed to parse skill index", "url", indexURL, "err", err)
		return nil
	}

	var filtered []IndexSkill
	for _, s := range idx.Skills {
		hasSkillMD := false
		for _, f := range s.Files {
			if strings.EqualFold(f, "SKILL.md") {
				hasSkillMD = true
				break
			}
		}
		if !hasSkillMD {
			d.Log.Warn("skill entry missing SKILL.md, skipping", "url", indexURL, "skill", s.Name)
			continue
		}
		filtered = append(filtered, s)
	}

	const skillConcurrency = 4
	sem := make(chan struct{}, skillConcurrency)
	var mu sync.Mutex
	var dirs []string

	var wg sync.WaitGroup
	for _, s := range filtered {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			root := filepath.Join(d.CacheDir, "skills", s.Name)

			const fileConcurrency = 8
			fileSem := make(chan struct{}, fileConcurrency)
			var fileWg sync.WaitGroup
			for _, f := range s.Files {
				f := f
				fileWg.Add(1)
				go func() {
					defer fileWg.Done()
					fileSem <- struct{}{}
					defer func() { <-fileSem }()

					dest := filepath.Join(root, f)
					fileURL := fmt.Sprintf("%s%s/%s", base, s.Name, f)
					d.download(fileURL, dest)
				}()
			}
			fileWg.Wait()

			md := filepath.Join(root, "SKILL.md")
			if _, err := os.Stat(md); err == nil {
				mu.Lock()
				dirs = append(dirs, root)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	return dirs
}

func (d *Discovery) download(fileURL, dest string) {
	if _, err := os.Stat(dest); err == nil {
		return
	}

	resp, err := d.Client.Get(fileURL)
	if err != nil {
		d.Log.Error("failed to download skill file", "url", fileURL, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		d.Log.Error("skill file download returned non-OK status", "url", fileURL, "status", resp.StatusCode)
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		d.Log.Error("failed to read skill file", "url", fileURL, "err", err)
		return
	}

	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		d.Log.Error("failed to create cache dir", "dir", dir, "err", err)
		return
	}

	if err := os.WriteFile(dest, data, 0o644); err != nil {
		d.Log.Error("failed to write skill file", "dest", dest, "err", err)
	}
}
