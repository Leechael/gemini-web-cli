package types

// AccountStatus represents the account status from the GetUserStatus RPC.
type AccountStatus struct {
	Code        int
	Name        string
	Description string
}

var (
	StatusAvailable                    = AccountStatus{1000, "AVAILABLE", "Account is authorized and has normal access."}
	StatusAccessTemporarilyUnavailable = AccountStatus{1014, "ACCESS_TEMPORARILY_UNAVAILABLE", "Access is restricted, possibly due to regional or temporary session issues."}
	StatusUnauthenticated              = AccountStatus{1016, "UNAUTHENTICATED", "Session is not authenticated or cookies have expired."}
	StatusAccountRejected              = AccountStatus{1021, "ACCOUNT_REJECTED", "Account access is rejected."}
	StatusAccountUntrusted             = AccountStatus{1033, "ACCOUNT_UNTRUSTED", "Account did not pass safety or trust checks for some features."}
	StatusTOSPending                   = AccountStatus{1040, "TOS_PENDING", "You need to accept the latest Terms of Service."}
	StatusTOSOutOfDate                 = AccountStatus{1042, "TOS_OUT_OF_DATE", "Terms of Service are out of date."}
	StatusAccountRejectedByGuardian    = AccountStatus{1054, "ACCOUNT_REJECTED_BY_GUARDIAN", "Access is blocked by a parent or guardian."}
	StatusGuardianApprovalRequired     = AccountStatus{1057, "GUARDIAN_APPROVAL_REQUIRED", "Access requires parent or guardian approval."}
	StatusLocationRejected             = AccountStatus{1060, "LOCATION_REJECTED", "Gemini is not currently supported in your country/region."}
)

// AccountStatusFromCode maps a numeric status code to an AccountStatus.
func AccountStatusFromCode(code int) AccountStatus {
	switch code {
	case 1000:
		return StatusAvailable
	case 1014:
		return StatusAccessTemporarilyUnavailable
	case 1016:
		return StatusUnauthenticated
	case 1021:
		return StatusAccountRejected
	case 1033:
		return StatusAccountUntrusted
	case 1040:
		return StatusTOSPending
	case 1042:
		return StatusTOSOutOfDate
	case 1054:
		return StatusAccountRejectedByGuardian
	case 1057:
		return StatusGuardianApprovalRequired
	case 1060:
		return StatusLocationRejected
	default:
		return AccountStatus{code, "UNKNOWN", "Unknown account status."}
	}
}

// IsHardBlock returns true for status codes that prevent any API usage.
func (s AccountStatus) IsHardBlock() bool {
	switch s.Code {
	case 1021, 1054, 1057, 1060, 1014:
		return true
	default:
		return false
	}
}
