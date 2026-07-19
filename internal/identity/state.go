package identity

// State は ISO/IEC 24760-1 が定義するアイデンティティのライフサイクル状態。
type State string

const (
	StateEnrolled  State = "enrolled"  // Enrolment: 登録
	StateIssued    State = "issued"    // Registration/Issuance: 発行
	StateActive    State = "active"    // Use/Maintenance: 利用・更新
	StateSuspended State = "suspended" // Suspension: 一時停止
	StateRevoked   State = "revoked"   // Revocation: 失効
	StateArchived  State = "archived"  // Archiving/Deletion: 保管・削除
)

// allowedTransitions は各状態から遷移可能な状態の一覧。
// Revoked -> Active のような復活は仕様上認めず、Archived は終端状態とする。
var allowedTransitions = map[State][]State{
	StateEnrolled:  {StateIssued},
	StateIssued:    {StateActive},
	StateActive:    {StateSuspended, StateRevoked},
	StateSuspended: {StateActive, StateRevoked},
	StateRevoked:   {StateArchived},
	StateArchived:  {},
}

// CanTransition は from から to への遷移が許可されているかを返す。
func CanTransition(from, to State) bool {
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}
