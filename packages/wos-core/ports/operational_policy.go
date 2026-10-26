package ports

// OperationalPolicyAuditUnitOfWork is the read snapshot required to assess an
// Outcome migration. It composes the existing repositories instead of adding a
// second storage model for Outcome state or work authority.
type OperationalPolicyAuditUnitOfWork interface {
	UnitOfWork
	ContinuityReadUnitOfWork
	WorkContractUnitOfWork
	SignedContractUnitOfWork
}
