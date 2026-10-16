package config

// GSTItemConfig defines official HSN code and GST rate for kirana commodities
type GSTItemConfig struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	HSNCode     string  `json:"hsn_code"`
	GSTRate     float64 `json:"gst_rate"`
	PackSize    string  `json:"pack_size"`
	Unit        string  `json:"unit"`
	BaseRate    float64 `json:"base_rate"`
}

// OfficialKiranaGSTSchedule defines standard GST rates for FMCG/Kirana commodities.
// Confirmed schedule:
// - Food staples (Atta, Dal, Edible Oil, Salt): 5%
// - Packaged Biscuits (Parle-G): 18%
// - Packaged Processed Foods (Noodles): 12%
// - Soaps, Detergents, Oral Care (Dettol, Surf Excel, Colgate): 18%
var OfficialKiranaGSTSchedule = []GSTItemConfig{
	{
		Category:    "Biscuits & Confectionery",
		Description: "Parle-G 12-pack Biscuit",
		HSNCode:     "1905",
		GSTRate:     18.0,
		PackSize:    "12-pack",
		Unit:        "pack",
		BaseRate:    118.00,
	},
	{
		Category:    "Salt & Minerals",
		Description: "Tata Salt Vacuum Evaporated 1kg",
		HSNCode:     "2501",
		GSTRate:     5.0,
		PackSize:    "1kg",
		Unit:        "kg",
		BaseRate:    24.00,
	},
	{
		Category:    "Edible Oils",
		Description: "Fortune Sunlite Sunflower Oil 1L",
		HSNCode:     "1512",
		GSTRate:     5.0,
		PackSize:    "1l",
		Unit:        "l",
		BaseRate:    135.00,
	},
	{
		Category:    "Food Staples & Grains",
		Description: "Aashirvaad Shudh Chakki Atta 10kg",
		HSNCode:     "1101",
		GSTRate:     5.0,
		PackSize:    "10kg",
		Unit:        "kg",
		BaseRate:    380.00,
	},
	{
		Category:    "Personal Hygiene & Soaps",
		Description: "Dettol Original Soap 75g",
		HSNCode:     "3401",
		GSTRate:     18.0,
		PackSize:    "75g",
		Unit:        "units",
		BaseRate:    38.00,
	},
	{
		Category:    "Packaged Instant Foods",
		Description: "Maggi 2-Minute Noodles Masala 70g",
		HSNCode:     "1902",
		GSTRate:     12.0,
		PackSize:    "70g",
		Unit:        "units",
		BaseRate:    14.00,
	},
	{
		Category:    "Fabric Care & Detergents",
		Description: "Surf Excel Quick Wash 1kg",
		HSNCode:     "3402",
		GSTRate:     18.0,
		PackSize:    "1kg",
		Unit:        "kg",
		BaseRate:    145.00,
	},
	{
		Category:    "Oral Care",
		Description: "Colgate Strong Teeth 200g",
		HSNCode:     "3306",
		GSTRate:     18.0,
		PackSize:    "200g",
		Unit:        "units",
		BaseRate:    110.00,
	},
}
