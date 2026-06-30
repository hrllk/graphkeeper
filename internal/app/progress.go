package app

import "hrllk/graphkeeper/internal/state"

func loadingToast(message string) state.Status {
	s := state.New().WithLoading(message)
	s.Detail = "Please wait."
	return s
}
