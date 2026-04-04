package dto

// ReplayStateResult reports replay rebuild counts and failures.
type ReplayStateResult struct {
	RebuiltCount int
	ErrorCount   int
}
