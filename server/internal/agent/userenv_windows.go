//go:build windows

package agent

import "golang.org/x/sys/windows/registry"

// readUserEnvironment is this Windows user's environment as it is now: what
// setx and System Properties write to HKCU\Environment. A process only sees the
// copy it inherited when it started, which is why this is read directly.
func readUserEnvironment() map[string]string {
	k, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, err := k.ReadValueNames(0)
	if err != nil {
		return nil
	}
	env := make(map[string]string, len(names))
	for _, name := range names {
		value, kind, err := k.GetStringValue(name)
		if err != nil {
			continue // not a string value, so not an environment variable
		}
		if kind == registry.EXPAND_SZ {
			if expanded, err := registry.ExpandString(value); err == nil {
				value = expanded
			}
		}
		env[name] = value
	}
	return env
}
