package ui

import "github.com/jroimartin/gocui"

type InputEditor struct {
	ui *UI
}

// https://github.com/djcas9/komanda-cli
func (e *InputEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
	case key == gocui.KeySpace:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyDelete:
		v.EditDelete(false)
	case key == gocui.KeyInsert:
		v.Overwrite = !v.Overwrite
	case key == gocui.KeyEnter:
		e.ui.sendMessage(e.ui.Gui, v)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		cx, _ := v.Cursor()
		line := v.ViewBuffer()
		if cx < len(line)-1 {
			v.MoveCursor(1, 0, false)
		}
	case key == gocui.KeyCtrlA:
		v.SetCursor(0, 0)
		v.SetOrigin(0, 0)
	case key == gocui.KeyCtrlK:
		v.Clear()
		v.SetCursor(0, 0)
		v.SetOrigin(0, 0)
	case key == gocui.KeyCtrlE:
		v.SetCursor(len(v.Buffer())-1, 0)
	}
}
