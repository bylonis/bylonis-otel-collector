package bylonisllmpricingprocessor // import "github.com/bylonis/bylonis-otel-collector/processor/bylonisllmpricingprocessor"

import (
	"fmt"
	"path"
	"time"
)

// DateLayout is the format of a rule's start date.
const DateLayout = "2006-01-02"

type CacheMode string

const (
	CacheModeSubtract CacheMode = "subtract" // cache_read is already counted inside input_tokens (e.g. OpenAI)
	CacheModeAdditive CacheMode = "additive" // cache_read is separate from input_tokens (e.g. Anthropic)
)

// Config is the top-level processor configuration.
type Config struct {
	// Attrs maps logical token-field names to the actual span attribute keys.
	Attrs AttrMapping `mapstructure:"attrs"`

	DefaultPricing PricingConfig `mapstructure:"default_pricing"`

	// OutputAttrs maps logical cost-field names to the span attribute keys
	// where computed costs are written.
	OutputAttrs OutputMapping `mapstructure:"output_attrs"`

	// SkipExisting leaves a span alone when it already has the total cost attribute. The AI Gateway
	// computes the cost at the source with the same table (#335); this keeps its figure and prices
	// only what other applications send without one.
	SkipExisting bool `mapstructure:"skip_existing"`
}

type AttrMapping struct {
	Model      string `mapstructure:"model"`
	In         string `mapstructure:"in"`
	Out        string `mapstructure:"out"`
	CacheRead  string `mapstructure:"cache_read"`
	CacheWrite string `mapstructure:"cache_write"`
	// Region is the span attribute that says where the model ran. Optional: without it, only rules
	// with no region apply.
	Region string `mapstructure:"region"`
}

type PricingConfig struct {
	Rules []PricingRule `mapstructure:"rules"`
}

type PricingRule struct {
	Name    string           `mapstructure:"name"`
	Pattern []string         `mapstructure:"pattern"`
	Cache   PricingRuleCache `mapstructure:"cache"`
	In      float64          `mapstructure:"in"`
	Out     float64          `mapstructure:"out"`

	// From is the date (YYYY-MM-DD) from which the rule applies. Empty: always. Several rules for the
	// same model with different dates are the history of its price: a span is priced with the rule that
	// was in force when it started (#646).
	From string `mapstructure:"from"`
	// Region restricts the rule to spans that report this region (see attrs.region). Empty: any region.
	Region string `mapstructure:"region"`
}

type PricingRuleCache struct {
	// Mode controls how cache tokens are factored into the cost:
	//   "subtract" — cache read tokens are already counted inside input_tokens;
	//                billed_input = input_tokens - cache_read.
	//   "additive" — cache read/write are separate from input_tokens;
	//                all four buckets are billed independently.
	Mode CacheMode `mapstructure:"mode"`

	// Per-million-token prices (USD) for cached reads/writes.
	Read  float64 `mapstructure:"read"`
	Write float64 `mapstructure:"write"`
}

type OutputMapping struct {
	In         string `mapstructure:"in"`
	Out        string `mapstructure:"out"`
	CacheRead  string `mapstructure:"cache_read"`
	CacheWrite string `mapstructure:"cache_write"`
	Total      string `mapstructure:"total"`
}

func (c *CacheMode) String() string {
	if c == nil {
		return ""
	}
	return string(*c)
}

func (c *Config) Validate() error {
	if c.Attrs.Model == "" {
		return fmt.Errorf("attrs.model must not be empty")
	}

	for i, r := range c.DefaultPricing.Rules {
		if len(r.Pattern) == 0 {
			return fmt.Errorf("default_pricing.rules[%d]: pattern must not be empty", i)
		}
		for j, p := range r.Pattern {
			if p == "" {
				return fmt.Errorf("default_pricing.rules[%d].pattern[%d]: pattern must not be empty", i, j)
			}
			if _, err := path.Match(p, ""); err != nil {
				return fmt.Errorf("default_pricing.rules[%d].pattern[%d]: invalid glob pattern %q: %w", i, j, p, err)
			}
		}
		if r.Cache.Mode != CacheModeSubtract && r.Cache.Mode != CacheModeAdditive {
			return fmt.Errorf("default_pricing.rules[%d] (pattern=%v): cache.mode must be %q or %q, got %q",
				i, r.Pattern, CacheModeSubtract, CacheModeAdditive, r.Cache.Mode)
		}
		if r.In < 0 || r.Out < 0 || r.Cache.Read < 0 || r.Cache.Write < 0 {
			return fmt.Errorf("default_pricing.rules[%d] (pattern=%v): prices must be non-negative", i, r.Pattern)
		}
		if r.From != "" {
			if _, err := time.Parse(DateLayout, r.From); err != nil {
				return fmt.Errorf("default_pricing.rules[%d] (pattern=%v): from %q is not a date YYYY-MM-DD", i, r.Pattern, r.From)
			}
		}
	}

	if c.OutputAttrs.Total == "" {
		return fmt.Errorf("output_attrs.total must not be empty")
	}

	return nil
}
