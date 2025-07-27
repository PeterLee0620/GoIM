package app

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
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
	contacts *Contacts
}

func NewApp(client *Client, contacts *Contacts) *App {
	app := tview.NewApplication()

	// -------------------------------------------------------------------------

	textview := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textview.SetBorder(true)
	textview.SetTitle(fmt.Sprintf("*** %s ***", contacts.My().ID))
	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")
	//清除文本 切换聊天界面
	list.SetChangedFunc(func(index int, name string, id string, shortcut rune) {
		textview.Clear()
		user, err := contacts.LookupContact(id)
		if err != nil {
			textview.ScrollToEnd()
			fmt.Fprintln(textview, "-----")
			fmt.Fprintln(textview, name+":"+err.Error())
			return
		}
		for i, msg := range user.Messages {
			fmt.Fprintln(textview, msg)
			if i < len(user.Messages)-1 {
				fmt.Fprintln(textview, "-----")
			}
		}
	})
	users := contacts.Contacts()
	for i, user := range users {
		shortcut := rune(i + 49)
		list.AddItem(user.Name, user.ID, shortcut, nil)
	}
	// -------------------------------------------------------------------------

	button := tview.NewButton("SUBMIT")
	button.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetBorder(true)
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
		contacts: contacts,
	}
	button.SetSelectedFunc(a.ButtonHandler)
	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			a.ButtonHandler()
			return nil
		}
		return event
	})

	return &a
}

func (a *App) Run() error {
	return a.app.SetRoot(a.flex, true).EnableMouse(true).Run()
}

func (a *App) WriteText(id string, msg string) {
	a.textview.ScrollToEnd()

	name, foundID := a.list.GetItemText(a.list.GetCurrentItem())
	if foundID == "" {
		fmt.Fprintln(a.textview, "-----")
		fmt.Fprintln(a.textview, "id not found"+":"+id)
		return
	}
	if foundID == id {
		fmt.Fprintln(a.textview, "-----")
		fmt.Fprintln(a.textview, name+":"+msg)
		return
	}

}

func (a *App) ButtonHandler() {
	_, to := a.list.GetItemText(a.list.GetCurrentItem())

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

func (a *App) UpdateContact(id string, name string) {
	shortcut := rune(a.list.GetItemCount() + 49)
	a.list.AddItem(name, id, shortcut, nil)
}
