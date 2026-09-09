package app

import "github.com/charmbracelet/lipgloss"

// The Inspector's two render paths disagreed about its own header. The screen
// path wrote "COMMIT <40 chars>" with the author's email and a "FROM <40
// chars>" tail on the author row; the popup path wrote "commit: <hash>" with
// neither. This file owns the copy decisions so there is one answer, and each
// path keeps its own fitting.
//
// The decisions, in the order they matter at 40 columns:
//
//   - A hash is shown whole or short, never cut. "4d8fcbcc16d09a9d81b2c3e4f50…"
//     reads as identity and is not one; the reader cannot use it to confirm
//     anything or to type it anywhere.
//   - The email is confirmation detail. The name identifies the author, so the
//     email is what goes when the row will not hold both.
//   - The parent gets a row of its own. As a tail on the author row it had no
//     matching TO, and it was the first thing cut -- at 80 columns, not just at
//     40.
//
// See docs/decisions.md 2026-08-03 for the priority these serve, and
// 2026-09-10 for this pass (task 6.8, D-016/D-018).

// inspectorShortHashWidth is the abbreviation the Inspector falls back to. It
// is longer than the graph's 5 because the Inspector is where a commit is
// confirmed rather than scanned.
const inspectorShortHashWidth = 12

// screenHashText steps a hash down to what the budget can hold whole, in the
// same shape as screenStampText. An empty result means the caller has no room
// to say anything truthful about the hash and should drop the row.
//
//	budget >= len(hash)  the full hash
//	budget >= 12         the first 12
//	budget >= 7          the first 7
//	otherwise            omitted
func screenHashText(hash string, budget int) string {
	if hash == "" || budget <= 0 {
		return ""
	}
	switch {
	case budget >= len(hash):
		return hash
	case budget >= inspectorShortHashWidth:
		return shorten(hash, inspectorShortHashWidth)
	case budget >= 7:
		return shorten(hash, 7)
	default:
		return ""
	}
}

// inspectorCommitText renders the identity row.
func inspectorCommitText(hash string, budget int) string {
	const key = "commit: "
	value := screenHashText(hash, budget-lipgloss.Width(key))
	if value == "" {
		return ""
	}
	return key + value
}

// inspectorAuthorText keeps the name and adds the email only when the row can
// hold it whole. A clipped domain ("<heykia3@protonmail.c…>") is a wrong
// address, not a shortened one.
func inspectorAuthorText(name, email string, budget int) string {
	const key = "author: "
	row := key + name
	if email == "" {
		return row
	}
	withEmail := row + " <" + email + ">"
	if lipgloss.Width(withEmail) <= budget {
		return withEmail
	}
	return row
}

// inspectorParentText renders the parent row. A root commit says so rather than
// leaving the reader to wonder why the row is missing.
func inspectorParentText(parent string, isRoot bool, budget int) string {
	const key = "parent: "
	if isRoot {
		return key + "(root commit)"
	}
	value := screenHashText(parent, budget-lipgloss.Width(key))
	if value == "" {
		return ""
	}
	return key + value
}
