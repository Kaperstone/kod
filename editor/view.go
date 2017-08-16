package editor

import (
	"github.com/gdamore/tcell"
	"github.com/linde12/kod/rpc"
)

const tabSize = 4

// TODO: Move this to RPC
type requestLines struct {
	Method string `json:"method"`
	Params []int  `json:"params"`
	ViewID string `json:"view_id"`
}

type View struct {
	*LineCache
	*InputHandler
	ID     string
	Editor *Editor
	ViewID string
	Width  int
	Height int

	// Topmost line in the text buffer, used for vertical scrolling
	Topline int
}

func NewView(path string, e *Editor) (*View, error) {
	view := &View{}
	view.Editor = e
	view.LineCache = NewLineCache()

	// fullscreen view
	w, h := e.screen.Size()
	view.Width = w
	view.Height = h

	msg, err := e.rpc.Request(&rpc.Request{
		Method: "new_view",
		Params: &rpc.Object{"file_path": path},
	})
	if err != nil {
		return view, err
	}

	view.ID = msg.Value.(string)
	view.InputHandler = &InputHandler{view.ID, path, e.rpc}

	// Set scroll window size
	e.rpc.Notify(&rpc.Request{
		Method: "edit",
		Params: &rpc.Object{
			"method":  "scroll",
			"params":  &rpc.Array{0, view.Height - 2},
			"view_id": view.ID,
		},
	})

	return view, nil
}

func (v *View) Draw() {
	if len(v.lines) == 0 {
		return
	}

	// TODO: Line numbers
	// TODO: Fix choppy scrolling
	for y, line := range v.lines[v.Topline:] {
		visualX := 0
		for _, char := range []rune(line.Text) {
			if char == '\t' {
				ts := tabSize - (visualX % tabSize)
				for i := 0; i < ts; i++ {
					v.Editor.screen.SetCell(visualX+i, y, v.Editor.defaultStyle, ' ')
				}
				visualX += ts
			} else {
				v.Editor.screen.SetCell(visualX, y, v.Editor.defaultStyle, char)
				visualX++
			}

			if len(line.Cursors) != 0 {
				// TODO: Verify if xi-core will take care of tabs for us
				cX := GetCursorVisualX(line.Cursors[0], line.Text)
				// TODO: Multiple cursor support
				v.Editor.screen.ShowCursor(cX, y)
			}
		}
	}
}

func (v *View) HandleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		if e.Key() == tcell.KeyRune {
			v.Insert(string(e.Rune()))
		} else {
			switch e.Key() {
			case tcell.KeyBackspace2, tcell.KeyBackspace:
				v.DeleteBackward()
			case tcell.KeyTAB:
				// TODO: Use v.Tab() when it's ready
				v.Insert("\t")
			case tcell.KeyEnter:
				v.Newline()
			case tcell.KeyLeft:
				v.MoveLeft()
			case tcell.KeyUp:
				v.MoveUp()
			case tcell.KeyRight:
				v.MoveRight()
			case tcell.KeyDown:
				v.MoveDown()
			case tcell.KeyCtrlQ:
				v.Editor.CloseView(v)
			case tcell.KeyCtrlS:
				v.Save()
			}
		}
	}
}
