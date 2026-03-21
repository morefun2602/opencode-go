package runtime

type Mode struct {
	Name string
	Tags []string
}

var (
	ModeBuild   = Mode{Name: "build", Tags: []string{"read", "write", "execute", "interact"}}
	ModePlan    = Mode{Name: "plan", Tags: []string{"read", "interact"}}
	ModeExplore = Mode{Name: "explore", Tags: []string{"read"}}
)

var builtinModes = map[string]Mode{
	"build":   ModeBuild,
	"plan":    ModePlan,
	"explore": ModeExplore,
}

func GetMode(name string) (Mode, bool) {
	m, ok := builtinModes[name]
	return m, ok
}

func RegisterMode(m Mode) {
	builtinModes[m.Name] = m
}

func ListModes() []Mode {
	out := make([]Mode, 0, len(builtinModes))
	for _, m := range builtinModes {
		out = append(out, m)
	}
	return out
}
