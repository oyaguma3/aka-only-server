package aka

import "fmt"

// SQNMask は SQN（48bit）のマスク。加算はこの範囲で折り返す。
const SQNMask = 1<<48 - 1

// SQNType は加入者ごとの SQN 増加タイプ。
type SQNType string

const (
	// SQNInc1 は SQN 全体を 1 加算する。
	SQNInc1 SQNType = "inc1"
	// SQNInc32 は SEQ を 1 加算し、IND は維持する。
	SQNInc32 SQNType = "inc32"
	// SQNInc33 は SEQ と IND を 1 ずつ加算する。複数生成時はバッチ内で IND を固定する。
	SQNInc33 SQNType = "inc33"
)

// Valid は t が定義済みの増加タイプかを返す。
func (t SQNType) Valid() bool {
	switch t {
	case SQNInc1, SQNInc32, SQNInc33:
		return true
	}
	return false
}

// Sequence は、最後に払い出した SQN が base のとき、次に払い出す n 個の SQN を返す。
// 払い出し後に保存する値は、返したスライスの最後の要素になる。
// Valkey 側の払い出しスクリプトと同じ計算でなければならない。
func (t SQNType) Sequence(base uint64, n int) ([]uint64, error) {
	var first, step uint64
	switch t {
	case SQNInc1:
		first, step = 1, 1
	case SQNInc32:
		first, step = 32, 32
	case SQNInc33:
		first, step = 33, 32
	default:
		return nil, fmt.Errorf("unknown SQN type %q", string(t))
	}
	sqns := make([]uint64, n)
	for i := range n {
		sqns[i] = (base + first + step*uint64(i)) & SQNMask
	}
	return sqns, nil
}
