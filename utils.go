package utils

type Status int

const(
	SellPlaced Status = iota	// 0
	SellCompleted				// 1
	TransferStarted				// 2
	TransferCompleted			// 3
	BuyPlaced					// 4
	BuyCompleted				// 5
)

func main() {

}

func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}
