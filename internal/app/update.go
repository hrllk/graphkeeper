package app

import tea "github.com/charmbracelet/bubbletea"

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return handleWindowSize(m, msg)
	case loadedMsg, refreshedMsg, tickMsg:
		return handleLifecycleUpdate(m, msg)
	case stashLoadedMsg:
		return handleStashUpdate(m, msg)
	case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, pushFetchedMsg, pullFetchedMsg, pullPreviewReadyMsg:
		return handleFetchUpdate(m, msg)
	case executedMsg:
		return handleExecutedUpdate(m, msg)
	case createdBranchMsg:
		return handleBranchUpdate(m, msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	default:
		return m, nil
	}
}
