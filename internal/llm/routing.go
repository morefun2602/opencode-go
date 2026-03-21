package llm

import (
	"fmt"
	"strings"
)

type ModelRef struct {
	ProviderID string
	ModelID    string
}

func ParseModel(s string) ModelRef {
	if s == "" {
		return ModelRef{}
	}
	idx := strings.Index(s, "/")
	if idx < 0 {
		return ModelRef{ModelID: s}
	}
	return ModelRef{
		ProviderID: s[:idx],
		ModelID:    s[idx+1:],
	}
}

var smallModelPriority = map[string][]string{
	"anthropic": {"claude-3-5-haiku-20241022", "claude-haiku-4-5-20250514"},
	"openai":    {"gpt-4o-mini", "gpt-4.1-mini"},
}

type Router struct {
	Registry     *Registry
	DefaultRef   ModelRef
	SmallRef     ModelRef
}

func NewRouter(reg *Registry, defaultModel, smallModel string) *Router {
	return &Router{
		Registry:   reg,
		DefaultRef: ParseModel(defaultModel),
		SmallRef:   ParseModel(smallModel),
	}
}

func (r *Router) DefaultModel() ModelRef {
	if r.DefaultRef.ModelID != "" {
		return r.DefaultRef
	}
	names := r.Registry.List()
	if len(names) == 0 {
		return ModelRef{}
	}
	prov, err := r.Registry.Get(names[0])
	if err != nil {
		return ModelRef{}
	}
	models := prov.Models()
	if len(models) == 0 {
		return ModelRef{ProviderID: names[0]}
	}
	return ModelRef{ProviderID: names[0], ModelID: models[0]}
}

func (r *Router) SmallModel() ModelRef {
	if r.SmallRef.ModelID != "" {
		return r.SmallRef
	}
	names := r.Registry.List()
	for _, name := range names {
		priorities, ok := smallModelPriority[name]
		if !ok {
			continue
		}
		prov, err := r.Registry.Get(name)
		if err != nil {
			continue
		}
		models := prov.Models()
		for _, wanted := range priorities {
			for _, available := range models {
				if strings.Contains(available, wanted) {
					return ModelRef{ProviderID: name, ModelID: available}
				}
			}
		}
		if len(models) > 0 {
			return ModelRef{ProviderID: name, ModelID: models[len(models)-1]}
		}
	}
	return r.DefaultModel()
}

func (r *Router) Resolve(ref ModelRef) (Provider, string, error) {
	if ref.ProviderID != "" {
		prov, err := r.Registry.Get(ref.ProviderID)
		if err != nil {
			return nil, "", err
		}
		model := ref.ModelID
		if model == "" {
			models := prov.Models()
			if len(models) > 0 {
				model = models[0]
			}
		}
		return prov, model, nil
	}

	if ref.ModelID != "" {
		for _, name := range r.Registry.List() {
			prov, err := r.Registry.Get(name)
			if err != nil {
				continue
			}
			for _, m := range prov.Models() {
				if m == ref.ModelID || strings.Contains(m, ref.ModelID) {
					return prov, m, nil
				}
			}
		}
	}

	return nil, "", fmt.Errorf("cannot resolve model: %+v", ref)
}

func (r *Router) ResolveDefault() (Provider, string, error) {
	return r.Resolve(r.DefaultModel())
}

func (r *Router) ResolveSmall() (Provider, string, error) {
	return r.Resolve(r.SmallModel())
}
