package commpackets

// 游戏布局是上黑下红
// 红方先手, 棋盘的大小是一半9*5, 共90个点
// 记左下角为0,0, 右上角为8,9
// 一个棋盘由9*10的二维数组组成, 这里给成90大小的一维数组
/*
y
上
|
|
|--------右 x
*/

// 棋盘是一个100大小的数组

type PieceType int

const (
	// 车
	PieceTypeChe PieceType = iota
	// 马
	PieceTypeMa
	// 象
	PieceTypeXiang
	// 士
	PieceTypeShi
	// 帅
	PieceTypeShuai
	// 炮
	PieceTypePao
	// 兵
	PieceTypeBing
)

type GameSide int

const (
	GameSideRed GameSide = iota
	GameSideBlack
)

type ChessPoint struct {
	X    int
	Y    int
	Type PieceType `json:"chess_piece_type"`
	Side GameSide  `json:"side"`
}

// 棋盘对象, 如果为null, 那么就是没有棋子
type ChessTable [90]*ChessPoint

func (ct ChessTable) ClearPoint(x int, y int) {
	ct[y*9+x] = nil
}

// 返回原来的棋子
func (ct *ChessTable) SetPoint(x int, y int, side GameSide, piece PieceType) *ChessPoint {
	former := ct[y*9+x]
	ct[y*9+x] = &ChessPoint{X: x, Y: y, Side: side, Type: piece}

	return former
}

// 修改拿到的点不会修改棋盘布局, 可以把拿到的对象占为己有
func (ct *ChessTable) GetPoint(x int, y int) *ChessPoint {
	p := ct[y*9+x]
	if p == nil {
		return nil
	}
	return &ChessPoint{X: x, Y: y, Type: p.Type, Side: p.Side}
}

// 返回默认布局的棋盘, 就是开始游戏的棋盘布局
func NewDefaultChessTable() *ChessTable {
	emptyTable := ChessTable{}

	// 红方
	emptyTable.SetPoint(0, 0, GameSideRed, PieceTypeChe)
	emptyTable.SetPoint(1, 0, GameSideRed, PieceTypeMa)
	emptyTable.SetPoint(2, 0, GameSideRed, PieceTypeXiang)
	emptyTable.SetPoint(3, 0, GameSideRed, PieceTypeShi)
	emptyTable.SetPoint(4, 0, GameSideRed, PieceTypeShuai)
	emptyTable.SetPoint(5, 0, GameSideRed, PieceTypeShi)
	emptyTable.SetPoint(6, 0, GameSideRed, PieceTypeXiang)
	emptyTable.SetPoint(7, 0, GameSideRed, PieceTypeMa)
	emptyTable.SetPoint(8, 0, GameSideRed, PieceTypeChe)
	emptyTable.SetPoint(1, 2, GameSideRed, PieceTypePao)
	emptyTable.SetPoint(7, 2, GameSideRed, PieceTypePao)
	emptyTable.SetPoint(0, 3, GameSideRed, PieceTypeBing)
	emptyTable.SetPoint(2, 3, GameSideRed, PieceTypeBing)
	emptyTable.SetPoint(4, 3, GameSideRed, PieceTypeBing)
	emptyTable.SetPoint(6, 3, GameSideRed, PieceTypeBing)
	emptyTable.SetPoint(8, 3, GameSideRed, PieceTypeBing)

	// 黑方
	emptyTable.SetPoint(0, 9, GameSideBlack, PieceTypeChe)
	emptyTable.SetPoint(1, 9, GameSideBlack, PieceTypeMa)
	emptyTable.SetPoint(2, 9, GameSideBlack, PieceTypeXiang)
	emptyTable.SetPoint(3, 9, GameSideBlack, PieceTypeShi)
	emptyTable.SetPoint(4, 9, GameSideBlack, PieceTypeShuai)
	emptyTable.SetPoint(5, 9, GameSideBlack, PieceTypeShi)
	emptyTable.SetPoint(6, 9, GameSideBlack, PieceTypeXiang)
	emptyTable.SetPoint(7, 9, GameSideBlack, PieceTypeMa)
	emptyTable.SetPoint(8, 9, GameSideBlack, PieceTypeChe)
	emptyTable.SetPoint(1, 7, GameSideBlack, PieceTypePao)
	emptyTable.SetPoint(7, 7, GameSideBlack, PieceTypePao)
	emptyTable.SetPoint(0, 6, GameSideBlack, PieceTypeBing)
	emptyTable.SetPoint(2, 6, GameSideBlack, PieceTypeBing)
	emptyTable.SetPoint(4, 6, GameSideBlack, PieceTypeBing)
	emptyTable.SetPoint(6, 6, GameSideBlack, PieceTypeBing)
	emptyTable.SetPoint(8, 6, GameSideBlack, PieceTypeBing)

	return &emptyTable
}
