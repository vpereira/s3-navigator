package main

import "github.com/rivo/tview"

// Display feedback in a modal
// The showModal function creates a modal dialog with a message and an OK button.
// For now it is connected with a form
func showModal(message string, returnToForm bool, form *tview.Form) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if returnToForm && form != nil {
				app.SetRoot(form, true)
			} else {
				app.SetRoot(mainLayout(), true)
			}
		})
	app.SetRoot(modal, true)
}
