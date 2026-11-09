package character

import "github.com/Whealinghold/srd5e-go/content"

// Barding is armor for a mount: four times the cost and twice the weight of
// the matching armor (SRD 5.2.1, "Barding", p. 100). It is an immutable value,
// comparable with ==, and keeps the printed coin of the armor's cost.
type Barding struct {
	cost   content.Cost
	weight content.Weight
}

// Cost returns four times the armor's cost, in the armor's coin.
func (b Barding) Cost() content.Cost { return b.cost }

// Weight returns twice the armor's weight, without the armor's note.
func (b Barding) Weight() content.Weight { return b.weight }

// DeriveBarding returns the barding for armor, a Shield included (it is on the
// Armor table). It returns an error matching srderr.ErrInvalidInput for an
// invalid armor (such as the zero Armor) or when four times the cost exceeds
// [content.MaxCostAmount]. The barding's Armor Class and which mount may wear
// it are not in the SRD and are not derived.
func DeriveBarding(armor content.Armor) (Barding, error) {
	price := armor.Price().Cost()
	if price.Amount() > content.MaxCostAmount/4 {
		return Barding{}, invalidInput("barding cost out of range")
	}
	cost, err := content.NewCost(price.Amount()*4, price.Coin(), 0)
	if err != nil {
		return Barding{}, err
	}
	weight, err := content.NewWeight(armor.Weight().Pounds()*2, "")
	if err != nil {
		return Barding{}, err
	}
	return Barding{cost: cost, weight: weight}, nil
}
