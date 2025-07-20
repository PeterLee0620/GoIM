package chat

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
	"github.com/rivo/tview"
)

type App struct {
	app      *tview.Application
	flex     *tview.Flex
	textview *tview.TextView
	button   *tview.Button
	client   *Client
	list     *tview.List
	textArea *tview.TextArea
}

func NewApp(client *Client) *App {
	app := tview.NewApplication()

	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")
	list.AddItem("Peter Lee", "f3cf4d43-9585-4398-8613-0a5787b1aede", '1', nil)
	list.AddItem("Keain Ardan", "c60e6de5-3b1d-4500-aba8-ca89903118d0", '2', nil)

	// -------------------------------------------------------------------------

	textview := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textview.SetBorder(true)
	textview.SetTitle("chat")

	// -------------------------------------------------------------------------

	button := tview.NewButton("SUBMIT")
	button.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetBorder(true)
	button.SetBorderPadding(0, 1, 0, 0)
	button.SetBorderColor(tcell.ColorGreen)
	button.SetSelectedFunc(func() {

	})
	// -------------------------------------------------------------------------

	textArea := tview.NewTextArea()
	textArea.SetWrap(false)
	textArea.SetPlaceholder("Enter message here...")
	textArea.SetBorder(true)
	textArea.SetBorderPadding(0, 0, 1, 0)

	// -------------------------------------------------------------------------

	flex := tview.NewFlex().
		AddItem(list, 20, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(textview, 0, 5, false).
			AddItem(tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(textArea, 0, 90, false).
				AddItem(button, 0, 10, false),
				0, 1, false),
			0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlQ:
			app.Stop()
			return nil
		}
		return event
	})

	a := App{
		app:      app,
		textview: textview,
		flex:     flex,
		button:   button,
		client:   client,
		list:     list,
		textArea: textArea,
	}
	button.SetSelectedFunc(a.ButtonHandler)
	return &a
}

func (a *App) Run() error {
	return a.app.SetRoot(a.flex, true).EnableMouse(true).Run()
}

func (a *App) WriteText(name string, msg string) {
	a.textview.ScrollToEnd()
	fmt.Fprintln(a.textview, "-----")
	fmt.Fprintln(a.textview, name+":"+msg)
}

func (a *App) ButtonHandler() {
	_, toIDStr := a.list.GetItemText(a.list.GetCurrentItem())
	to, err := uuid.Parse(toIDStr)
	if err != nil {
		a.WriteText("system", fmt.Sprintf("Error parse UUID:%s", err))
		return
	}

	msg := a.textArea.GetText()
	if msg == "" {
		return
	}
	if err := a.client.Send(to, msg); err != nil {
		a.WriteText("system", fmt.Sprintf("Error Send msg:%s", err))
		return
	}
	a.textArea.SetText("", false)
	a.WriteText("You", msg)
}

func (a *App) FindName(id string) string {
	for i := 0; i < a.list.GetItemCount(); i++ {
		name, toIDStr := a.list.GetItemText(i)
		if id == toIDStr {
			return name
		}
	}

	return ""
}
