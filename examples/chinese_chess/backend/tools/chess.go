package tools

import (
	"math"

	commpackets "github.com/markity/go-reactor/examples/chinese_chess/backend/common_packets"
)

func isCrossedRiver(side commpackets.GameSide, x, y int) bool {
	if side == commpackets.GameSideRed {
		return y >= 5
	} else {
		return y <= 4
	}
}

func pointMatch(x1 int, y1 int, x2 int, y2 int) bool {
	return x1 == x2 && y1 == y2
}

func isPointCrossRange(x, y int) bool {
	return x > 8 || y > 9 || x < 0 || y < 0
}

func isSameLine(x1, y1, x2, y2 int) bool {
	return x1 == x2 || y1 == y2
}

// 检查两个点之间有没有点, 假设已经在一条直线上, 两点不能是同一点
// 不包含两个点, 如果要判断端点, 自行加判断
func hasChessBetweenTwoPoints(table *commpackets.ChessTable, x1 int, y1 int, x2 int, y2 int) bool {
	if x1 == x2 {
		var yMin int
		var yMax int
		if y1 < y2 {
			yMin = y1
			yMax = y2
		} else {
			yMin = y2
			yMax = y1
		}

		for y0 := yMin + 1; y0 < yMax; y0++ {
			if table.GetPoint(x1, y0) != nil {
				return true
			}
		}
	} else {
		var xMin int
		var xMax int
		if x1 < x2 {
			xMin = x1
			xMax = x2
		} else {
			xMin = x2
			xMax = x1
		}

		for x0 := xMin + 1; x0 < xMax; x0++ {
			if table.GetPoint(x0, y1) != nil {
				return true
			}
		}
	}
	return false

}

func DoMove(side commpackets.GameSide, table *commpackets.ChessTable, fromX int, fromY int, toX int, toY int) (gameover bool, moveok bool) {
	// 不能有非法的坐标
	if isPointCrossRange(fromX, fromY) || isPointCrossRange(toX, toY) {
		return false, false
	}

	// from点不能是空的
	from := table.GetPoint(fromX, fromY)
	if from == nil {
		return false, false
	}

	// 不能移动非己方的棋子
	if from.Side != side {
		return false, false
	}

	// 不能不移动
	if from.X == toX && from.Y == toY {
		return false, false
	}

	switch from.Type {
	case commpackets.PieceTypeBing:
		if isCrossedRiver(side, from.X, from.Y) {
			// 过河了, 移动方向只能是上下左右
			if !pointMatch(toX, toY, from.X+1, from.Y) &&
				!pointMatch(toX, toY, from.X, from.Y+1) &&
				!pointMatch(toX, toY, from.X-1, from.Y) &&
				!pointMatch(toX, toY, from.X, from.Y-1) {
				return false, false
			}

			// 红方的y不能减小
			if side == commpackets.GameSideRed {
				if from.Y > toY {
					return false, false
				}
			} else {
				// 黑方的y不能增大
				if from.Y < toY {
					return false, false
				}
			}
		} else {
			// 没有过河, 只能前后移动
			if !pointMatch(toX, toY, from.X, from.Y+1) &&
				!pointMatch(toX, toY, from.X, from.Y-1) {
				return false, false
			}

			// 红方的y不能减小
			if side == commpackets.GameSideRed {
				if from.Y > toY {
					return false, false
				}
			} else {
				// 黑方的y不能增大
				if from.Y < toY {
					return false, false
				}
			}
		}
		table.ClearPoint(fromX, fromY)
		pre := table.SetPoint(toX, toY, side, commpackets.PieceTypeBing)
		return pre != nil && pre.Type == commpackets.PieceTypeShuai, true
	case commpackets.PieceTypePao:
		// 炮至少要走直线
		if !isSameLine(from.X, from.Y, toX, toY) {
			return false, false
		}

		// 如果两个点中间没有子
		if !hasChessBetweenTwoPoints(table, from.X, from.Y, toX, toY) {
			// 如果终点有子, 失败
			if table.GetPoint(toX, toY) != nil {
				return false, false
			}
		} else {
			// 如果中间有子, 判断对面是否是敌方的子
			if table.GetPoint(toX, toY).Side == side {
				return false, false
			}
		}
		table.ClearPoint(from.X, from.Y)
		pre := table.SetPoint(toX, toY, side, commpackets.PieceTypePao)
		return pre != nil && pre.Type == commpackets.PieceTypeShuai, true
	case commpackets.PieceTypeChe:
		// 必须是直线
		if !isSameLine(from.X, from.Y, toX, toY) {
			return false, false
		}

		// 中间如果有子
		if hasChessBetweenTwoPoints(table, from.X, from.Y, toX, toY) {
			return false, false
		}

		// 直线, 中间无子, 对端有子是己方阵营
		if table.GetPoint(toX, toY) != nil {
			if table.GetPoint(toX, toY).Side == side {
				return false, false
			}
		}

		table.ClearPoint(from.X, from.Y)
		pre := table.SetPoint(toX, toY, side, commpackets.PieceTypeChe)
		return pre != nil && pre.Type == commpackets.PieceTypeShuai, true
	case commpackets.PieceTypeMa:
		if !pointMatch(toX, toY, from.X+1, from.Y+2) &&
			!pointMatch(toX, toY, from.X+1, from.Y-2) &&
			!pointMatch(toX, toY, from.X+2, from.Y+1) &&
			!pointMatch(toX, toY, from.X+2, from.Y-1) &&
			!pointMatch(toX, toY, from.X-2, from.Y+1) &&
			!pointMatch(toX, toY, from.X-2, from.Y-1) &&
			!pointMatch(toX, toY, from.X-1, from.Y+2) &&
			!pointMatch(toX, toY, from.X-1, from.Y-2) {
			return false, false
		}

		var pX, pY int

		if from.X-toX == -2 {
			pX = from.X + 1
			pY = from.Y
		}
		if from.X-toX == 2 {
			pX = from.X - 1
			pY = from.Y
		}
		if from.Y-toY == -2 {
			pX = from.X
			pY = from.Y + 1
		}
		if from.Y-toY == 2 {
			pX = from.X
			pY = from.Y - 1
		}

		if table.GetPoint(pX, pY) != nil {
			return false, false
		}

		table.ClearPoint(from.X, from.Y)
		pre := table.SetPoint(toX, toY, side, commpackets.PieceTypeMa)
		return pre != nil && pre.Type == commpackets.PieceTypeShuai, true
	case commpackets.PieceTypeXiang:
		if !pointMatch(toX, toY, from.X+2, from.Y+2) &&
			!pointMatch(toX, toY, from.X+2, from.Y-2) &&
			!pointMatch(toX, toY, from.X-2, from.Y+2) &&
			!pointMatch(toX, toY, from.X-2, from.Y-2) {
			return false, false
		}

		// 象不能过河
		if isCrossedRiver(side, toX, toY) {
			return false, false
		}

		diffX := from.X - toX
		diffY := from.Y - toY
		pX := from.X
		pY := from.Y

		if diffX == -2 {
			pX++
		}
		if diffX == 2 {
			pX--
		}
		if diffY == -2 {
			pY++
		}
		if diffY == 2 {
			pY--
		}

		// 不能堵象眼
		if table.GetPoint(pX, pY) != nil {
			return false, false
		}

		table.ClearPoint(from.X, from.Y)
		pre := table.SetPoint(toX, toY, side, commpackets.PieceTypeXiang)
		return pre != nil && pre.Type == commpackets.PieceTypeShuai, true
	case commpackets.PieceTypeShi:
		// 必须出现在特定的位置
		if side == commpackets.GameSideRed {
			if !pointMatch(toX, toY, 3, 0) &&
				!pointMatch(toX, toY, 5, 0) &&
				!pointMatch(toX, toY, 4, 1) &&
				!pointMatch(toX, toY, 3, 2) &&
				!pointMatch(toX, toY, 5, 2) {
				return false, false
			}
		} else {
			if !pointMatch(toX, toY, 3, 9) &&
				!pointMatch(toX, toY, 5, 9) &&
				!pointMatch(toX, toY, 4, 8) &&
				!pointMatch(toX, toY, 3, 7) &&
				!pointMatch(toX, toY, 5, 7) {
				return false, false
			}
		}

		// 不能限制移动的位置
		diffX := from.X - toX
		diffY := from.Y - toY
		if diffX > 1 || diffX < -1 {
			return false, false
		}
		if diffY > 1 || diffY < -1 {
			return false, false
		}

		table.ClearPoint(from.X, from.Y)
		pre := table.SetPoint(toX, toY, side, commpackets.PieceTypeShi)
		return pre != nil && pre.Type == commpackets.PieceTypeShuai, true
	case commpackets.PieceTypeShuai:
		// 特殊规则, 如果帅与将之间没有间隔的东西
		if table.GetPoint(toX, toY) != nil && table.GetPoint(toX, toY).Type == commpackets.PieceTypeShuai &&
			isSameLine(from.X, from.Y, toX, toY) &&
			!hasChessBetweenTwoPoints(table, from.X, from.Y, toX, toY) {
			table.ClearPoint(from.X, from.Y)
			table.SetPoint(toX, toY, side, commpackets.PieceTypeShuai)
			return true, true
		}

		// 一般规则

		// 要求to在九宫格之内
		if side == commpackets.GameSideRed {
			if toX < 3 || toX > 5 || toY > 2 {
				return false, false
			}
		} else {
			if toX < 3 || toX > 5 || toY < 7 {
				return false, false
			}
		}

		// 要求只能走一格
		diffX := int(math.Abs(float64(from.X) - float64(toX)))
		diffY := int(math.Abs(float64(from.Y) - float64(toY)))
		if diffX+diffY != 1 {
			return false, false
		}

		// 要求to的地方没有自己的子
		if table.GetPoint(toX, toY) != nil && table.GetPoint(toX, toY).Side == side {
			return false, false
		}

		table.ClearPoint(fromX, fromY)
		table.SetPoint(toX, toY, side, commpackets.PieceTypeShuai)
		return false, true
	}

	// unreachable
	return false, false
}
