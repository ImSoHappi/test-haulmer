package domain

// Deterministic test PAN constants.
const (
	PANApproved          = "4242424242424242"
	PANInsufficientFunds = "4000000000009995"
	PANAcquirerDown      = "4000000000000119"
	PANFlakyAcquirer     = "4000000000000341"
	PANTimeoutNoAuth     = "4000000000000101"
	PANTimeoutWithAuth   = "4000000000000259"
)

// CardBehavior represents the simulated behavior for a specific card.
type CardBehavior string

const (
	BehaviorApproved                 CardBehavior = "APPROVED"
	BehaviorDeclinedInsufficientFunds CardBehavior = "DECLINED_INSUFFICIENT_FUNDS"
	BehaviorPersistent503            CardBehavior = "PERSISTENT_503"
	BehaviorFlaky503                 CardBehavior = "FLAKY_503"
	BehaviorTimeoutNoAuth            CardBehavior = "TIMEOUT_NO_AUTH"
	BehaviorTimeoutWithAuth          CardBehavior = "TIMEOUT_WITH_AUTH"
)

// ClassifyCard maps a PAN to its deterministic simulation behavior.
// Any valid PAN not explicitly listed defaults to BehaviorApproved.
func ClassifyCard(pan string) CardBehavior {
	switch pan {
	case PANInsufficientFunds:
		return BehaviorDeclinedInsufficientFunds
	case PANAcquirerDown:
		return BehaviorPersistent503
	case PANFlakyAcquirer:
		return BehaviorFlaky503
	case PANTimeoutNoAuth:
		return BehaviorTimeoutNoAuth
	case PANTimeoutWithAuth:
		return BehaviorTimeoutWithAuth
	case PANApproved:
		fallthrough
	default:
		return BehaviorApproved
	}
}
