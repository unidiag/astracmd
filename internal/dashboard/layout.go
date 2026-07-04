package dashboard

import "github.com/rivo/tview"

func centerPrimitive(p tview.Primitive, width int, height int) tview.Primitive {
	row := tview.NewFlex()
	row.SetDirection(tview.FlexRow)
	row.AddItem(nil, 0, 1, false)
	row.AddItem(p, height, 1, true)
	row.AddItem(nil, 0, 1, false)

	root := tview.NewFlex()
	root.SetDirection(tview.FlexColumn)
	root.AddItem(nil, 0, 1, false)
	root.AddItem(row, width, 1, true)
	root.AddItem(nil, 0, 1, false)

	return root
}
