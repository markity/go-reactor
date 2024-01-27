package tools

import (
	"fmt"

	commpackets "github.com/markity/go-reactor/examples/chinese_chess/frontend/comm_packets"

	interactive "github.com/markity/Interactive-Console"
)

func getPieceTypeName(point *commpackets.ChessPoint) string {
	if point == nil {
		return "  "
	}
	switch point.Type {
	case commpackets.PieceTypeBing:
		return "兵"
	case commpackets.PieceTypeChe:
		return "车"
	case commpackets.PieceTypeMa:
		return "马"
	case commpackets.PieceTypePao:
		return "炮"
	case commpackets.PieceTypeShi:
		return "士"
	case commpackets.PieceTypeShuai:
		return "帅"
	case commpackets.PieceTypeXiang:
		return "象"
	default:
		panic("check here")
	}
}

func DrawTable(table commpackets.ChessTable, win *interactive.Win, msg string) {
	win.Clear()
	win.SendLineBackWithColor("  0 1 2 3 4 5 6 7 8")
	for y := 9; y >= 0; y-- {
		sendbuf := make([]interface{}, 0)
		sendbuf = append(sendbuf, fmt.Sprintf("%d ", y))
		for x := 0; x <= 8; x++ {
			name := getPieceTypeName(table.GetPoint(x, y))
			if table.GetPoint(x, y) != nil {
				if table.GetPoint(x, y).Side == commpackets.GameSideRed {
					color := interactive.GetDefaultSytleAttr()
					color.Foreground = interactive.ColorRed
					sendbuf = append(sendbuf, color)
				} else {
					color := interactive.GetDefaultSytleAttr()
					color.Foreground = interactive.ColorBlue
					sendbuf = append(sendbuf, color)
				}
			}
			sendbuf = append(sendbuf, name)
		}
		sendbuf = append(sendbuf, interactive.GetDefaultSytleAttr(), fmt.Sprint(y))
		win.SendLineBackWithColor(sendbuf...)
		if y == 5 {
			riverColor := interactive.GetDefaultSytleAttr()
			riverColor.Foreground = interactive.ColorDarkTurquoise
			win.SendLineBackWithColor(riverColor, "========楚河=========")
		}
	}
	win.SendLineBackWithColor("  0 1 2 3 4 5 6 7 8")
	win.SendLineBack(msg)
}
