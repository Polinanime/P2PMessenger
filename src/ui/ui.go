package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/polinanime/p2pmessenger/models"
)

const (
	timestampFormatServer = "2006-01-02 15:04:05.999999999 -0700 MST"
)

type UI struct {
	*gocui.Gui
	messenger *models.P2PMessenger
	users     []map[string]string
}

func NewUI(p *models.P2PMessenger) (*UI, error) {
	gui, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, fmt.Errorf("NewUI: %w", err)
	}

	users := p.GetUsers()
	return &UI{Gui: gui, messenger: p, users: users}, nil
}

func (ui *UI) StartGui() {
	defer ui.Close()

	ui.SetManagerFunc(ui.layout)

	if err := ui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := ui.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, ui.refresh); err != nil {
		log.Panicln(err)
	}

	if err := ui.SetKeybinding("users", gocui.KeyArrowDown, gocui.ModNone, ui.cursorDown); err != nil {
		log.Panicln(err)
	}

	if err := ui.SetKeybinding("users", gocui.KeyArrowUp, gocui.ModNone, ui.cursorUp); err != nil {
		log.Panicln(err)
	}

	if err := ui.SetKeybinding("users", gocui.KeyEnter, gocui.ModNone, ui.selectUser); err != nil {
		log.Panicln(err)
	}

	if err := ui.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, ui.sendMessage); err != nil {
		log.Panicln(err)
	}

	// if err := ui.SetKeybinding("input", gocui.KeySpace, gocui.ModNone, ui.handleSpace); err != nil {
	// 	log.Panicln(err)
	// }

	// Switch between views
	if err := ui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, ui.nextView); err != nil {
		log.Panicln(err)
	}

	if err := ui.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func (ui *UI) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("header", 0, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "P2P Messenger"
		fmt.Fprintln(v, "Press Ctrl+C to quit, Ctrl+R to refresh")
	}

	if v, err := g.SetView("users", 0, 3, maxX/4-1, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Users"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Wrap = false
		ui.updateUsersView(v)
	}

	if v, err := g.SetView("chat", maxX/4, 3, maxX-1, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Chat"
	}

	if v, err := g.SetView("input", 0, maxY-4, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Message Input"
		v.Editable = true
		v.Wrap = true
		v.Clear()
		v.Editor = &InputEditor{ui: ui}
	}

	if _, err := g.SetCurrentView("users"); err != nil {
		return err
	}

	return nil
}

func (ui *UI) cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if cy >= len(ui.users)-1 {
			return nil
		}
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ui *UI) cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if cy == 0 {
			return nil
		}
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ui *UI) nextView(g *gocui.Gui, v *gocui.View) error {
	currentView := g.CurrentView()
	if currentView == nil {
		_, err := g.SetCurrentView("users")
		return err
	}

	switch currentView.Name() {
	case "users":
		_, err := g.SetCurrentView("input")
		return err
	case "input":
		_, err := g.SetCurrentView("users")
		return err
	default:
		_, err := g.SetCurrentView("users")
		return err
	}
}

func (ui *UI) updateUsersView(v *gocui.View) {
	v.Clear()
	ui.users = ui.messenger.GetUsers()
	for _, user := range ui.users {
		if userID, ok := user["userID"]; ok {
			fmt.Fprintf(v, "%s\n", userID)
		}
	}
}

func (ui *UI) selectUser(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	userID, err := v.Line(cy)
	if err != nil || userID == "" {
		return nil
	}

	chatView, err := g.View("chat")
	if err != nil {
		return err
	}
	ui.updateChatView(chatView, userID)

	inputView, err := g.View("input")
	if err != nil {
		return err
	}
	inputView.Clear()
	fmt.Fprintf(inputView, "To: %s\n", userID)

	inputView.Editable = true
	// Set cursot to input view
	inputView.SetCursor(0, 1)
	// Set focus to input view
	if _, err := g.SetCurrentView("input"); err != nil {
		return err
	}
	log.Printf("Current view: %s", g.CurrentView().Name())

	return nil
}

func (ui *UI) updateChatView(v *gocui.View, userID string) {
	v.Clear()
	messages := ui.messenger.GetHistory()
	for _, msg := range messages {
		if msg.From == userID || msg.To == userID {
			fmt.Fprintf(v, "[%s] %s -> %s: %s\n",
				msg.Timestamp.Format(time.RFC822),
				msg.From,
				msg.To,
				msg.Content)
		}
	}
}

func (ui *UI) sendMessage(g *gocui.Gui, v *gocui.View) error {
	input := v.Buffer()
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) < 2 {
		return nil
	}

	to := strings.TrimSpace(strings.TrimPrefix(lines[0], "To:"))
	message := strings.TrimSpace(strings.Join(lines[1:], "\n"))

	if message == "" {
		return nil
	}

	if err := ui.messenger.SendMessage(to, message); err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}

	v.Clear()
	fmt.Fprintf(v, "To: %s\n", to)

	v.SetCursor(0, 1)

	chatView, err := g.View("chat")
	if err != nil {
		return err
	}
	ui.updateChatView(chatView, to)

	return nil
}

func (ui *UI) updateMessagesView(v *gocui.View) {
	v.Clear()
	messages := ui.messenger.GetHistory()
	for _, msg := range messages {
		fmt.Fprintf(v, "[%s] %s -> %s: %s\n",
			msg.Timestamp.Format(time.RFC822),
			msg.From,
			msg.To,
			msg.Content)
	}
}

func (ui *UI) refresh(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()

	// Create the loading view
	loadingView, err := g.SetView("loading", maxX/2-10, maxY/2-1, maxX/2+10, maxY/2+1)
	ui.SetViewOnTop("loading")
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	loadingView.Title = "Loading"
	loadingView.Wrap = true
	loadingView.Frame = true
	loadingView.Clear()
	fmt.Fprintln(loadingView, "Refreshing...")

	go func() {
		defer func() {
			g.Update(func(g *gocui.Gui) error {
				// Delete the loading view
				if err := g.DeleteView("loading"); err != nil {
					return err
				}
				return nil
			})
		}()

		usersView, err := g.View("users")
		if err != nil {
			log.Println(err)
			return
		}
		ui.updateUsersView(usersView)

		chatView, err := g.View("chat")
		if err != nil {
			log.Println(err)
			return
		}
		// Get the current user ID from the input view to refresh the chat view
		inputView, err := g.View("input")
		if err != nil {
			log.Println(err)
			return
		}
		lines := strings.Split(inputView.Buffer(), "\n")
		if len(lines) > 0 {
			to := strings.TrimPrefix(lines[0], "To: ")
			ui.updateChatView(chatView, to)
		}
	}()

	if _, err := g.SetCurrentView("input"); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
