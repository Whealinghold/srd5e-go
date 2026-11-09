package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
)

// A Fighter takes the Alert feat through a custom background's Origin feat slot;
// the result lists the FeatTaken event, and the declared effects of the feat follow
// from the feats held.
func ExampleTakeFeat() {
	alert := content.MustFeat("alert", "Alert", content.SourceCitation{}, content.FeatFields{
		Category: content.FeatOrigin,
		Benefits: []content.FeatBenefit{content.MustFeatBenefit("Initiative Proficiency", "Add your Proficiency Bonus to Initiative.")},
	})
	use, _ := character.NewTakeFeat(contenttest.NewFakeSource(alert))
	grant, _ := character.NewFeatGrant(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
	res, _ := use.Run(context.Background(), character.TakeFeatCommand{
		Character: pcExample(), Feat: "alert", Grant: grant, Choices: character.NoFeatChoices{},
	})
	taken := res.(character.TakeFeatSucceeded)
	fmt.Println(taken.Events()[0].(character.FeatTaken).Feat(), taken.Feats().Holds("alert"), len((character.FeatEffects{}).Effects(taken.Feats())))
	// Output: alert true 2
}
