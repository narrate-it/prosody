package briefing

import "reflect"

// missingProvider rejects nil interface values and nil underlying ports before
// accounting or handoff. It does not recover provider panics: after handoff,
// failures retain their existing conservative provider-attempt semantics.
func missingProvider(provider any) bool {
	if provider == nil {
		return true
	}
	value := reflect.ValueOf(provider)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
