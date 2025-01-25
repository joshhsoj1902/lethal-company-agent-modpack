package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"

	// "regexp"
	"strings"
)

type ItemConfig struct {
	Name           string
	MoonRarity     map[string]int
	ScrapValueMin  float32
	ScrapValueMax  float32
	IsMetal        bool
	Weight         float32
	Battery        string
	ShopPrice      string
	TwoHanded      bool
	CarryEffect    string
	IsShop         bool
	IsBody         bool
	Raw            []string
	MinWeightRatio float32
	MaxWeightRatio float32
	ComputedRarity float32
}

type RebuiltConfig struct {
	Items map[string]ItemConfig
}

func parseAgentConfig() RebuiltConfig {
	/////////////////////
	// Parse Agent Config
	/////////////////////

	file, err := os.Open("BepInEx/config/ConfigurableCompany/Presets/Agent.ccfg")
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var rebuiltConfig RebuiltConfig
	rebuiltConfig.Items = make(map[string]ItemConfig)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimPrefix(line, "[")
		line = strings.TrimSuffix(line, "]")

		switch {
		case strings.HasPrefix(line, "lcv_conifg-"):
			parts := strings.Split(line, "-")
			moon := parts[1]
			parts2 := strings.Split(parts[3], ":")
			name := parts2[0]

			switch parts[2] {
			case "item":
				itemConfig, exists := rebuiltConfig.Items[name]
				if !exists {
					itemConfig = ItemConfig{
						Name: name,
					}
					itemConfig.MoonRarity = make(map[string]int)
				}

				itemConfig.Raw = append(rebuiltConfig.Items[name].Raw, line)

				// Parse the rarity for that moon
				parts3 := strings.Split(parts2[1], "|")

				rarity, err := strconv.Atoi(parts3[0])
				if err != nil {
					fmt.Printf("Error parsing rarity: %s\n", err)
					continue
				}
				itemConfig.MoonRarity[moon] = rarity

				rebuiltConfig.Items[name] = itemConfig
			}
		case strings.HasPrefix(line, "lcv_config_item"):
			parts := strings.Split(line, "|")

			name := ""
			minValue := ""
			maxValue := ""

			trimmed := strings.TrimPrefix(parts[0], "lcv_config_item-")
			parts = strings.Split(trimmed, "_")
			category := parts[0]

			switch len(parts) {
			case 2:
				// the format in this case looks like `lcv_config_item-metal_cassettenazareitem:false|enabled:False`
				parts = strings.Split(parts[1], ":")
				name = parts[0]
				if len(parts) == 2 {
					minValue = parts[1]
					maxValue = parts[1]
				} else {
					minValue = strings.TrimSuffix(parts[1], "\\")
					maxValue = parts[2]
				}

			case 3, 4, 5:
				// the format in this case looks like `lcv_config_item-weight_control_panel_item:5.25|enabled:False`
				name = strings.Join(parts[1:len(parts)-1], "_")
				parts = strings.Split(parts[len(parts)-1], ":")
				if len(parts) == 2 {
					minValue = parts[1]
					maxValue = parts[1]
				} else {
					minValue = strings.TrimSuffix(parts[1], "\\")
					maxValue = parts[2]
				}
			default:
				fmt.Printf("Line: %s\n", line)
				fmt.Printf("Unexpected number of parts when parsing category: %+v\n", parts)
				continue
			}

			itemConfig, exists := rebuiltConfig.Items[name]
			if !exists {
				itemConfig = ItemConfig{
					Name: name,
				}
				itemConfig.MoonRarity = make(map[string]int)
			}

			switch category {
			case "weight":
				val, err := strconv.ParseFloat(minValue, 32)
				if err != nil {
					fmt.Printf("Line: %s\n", line)
					fmt.Printf("Error parsing weight: %s\n", err)
					continue
				}
				itemConfig.Weight = float32(val)
			case "scrap-worth":
				val, err := strconv.ParseFloat(minValue, 32)
				if err != nil {
					fmt.Printf("Line: %s\n", line)
					fmt.Printf("Error parsing scrap-worth: %s\n", err)
					continue
				}
				itemConfig.ScrapValueMin = float32(val)
				val, err = strconv.ParseFloat(maxValue, 32)
				if err != nil {
					fmt.Printf("Line: %s\n", line)
					fmt.Printf("Error parsing scrap-worth: %s\n", err)
					continue
				}
				itemConfig.ScrapValueMax = float32(val)
			case "metal":
				// Cast string to bool
				itemConfig.IsMetal = minValue == "true"
			case "battery":
				itemConfig.Battery = minValue
			case "shop-price":
				itemConfig.ShopPrice = minValue
				itemConfig.IsShop = true
			default:
				fmt.Printf("Line: %s\n", line)
				fmt.Printf("Unknown category: %s\n", category)
			}
			rebuiltConfig.Items[name] = itemConfig
		}
	}

	return rebuiltConfig
}

func parseScrapHardcoded(rebuiltConfig RebuiltConfig) RebuiltConfig {
	//////////////////////////
	// Parse Override Settings
	//////////////////////////

	file, err := os.Open("scrapHardcoded.data")
	if err != nil {
		fmt.Println("Error opening scrapHardcoded.data:", err)
		os.Exit(2)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	isFirstLine := true
	for scanner.Scan() {
		if isFirstLine {
			isFirstLine = false
			continue
		}

		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue // Skip comments and empty lines
		}

		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			fmt.Printf("Invalid line format in scrapHardcoded.data: %s\n", line)
			continue
		}

		name := strings.TrimSpace(parts[0])
		itemConfig, exists := rebuiltConfig.Items[name]
		if !exists {
			itemConfig = ItemConfig{
				Name:       name,
				MoonRarity: make(map[string]int),
			}
		}

		// Update two-handed and carry effect
		itemConfig.TwoHanded = strings.TrimSpace(parts[1]) == "true"
		itemConfig.CarryEffect = strings.TrimSpace(parts[2])
		itemConfig.IsBody = strings.TrimSpace(parts[4]) == "true"

		rebuiltConfig.Items[name] = itemConfig
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading scrapHardcoded.data:", err)
	}

	return rebuiltConfig
}

func calculateWeightRatios(rebuiltConfig RebuiltConfig) RebuiltConfig {
	for _, itemConfig := range rebuiltConfig.Items {
		itemConfig.MinWeightRatio = itemConfig.ScrapValueMin / itemConfig.Weight
		itemConfig.MaxWeightRatio = itemConfig.ScrapValueMax / itemConfig.Weight
		rebuiltConfig.Items[itemConfig.Name] = itemConfig
	}

	return rebuiltConfig
}

func calculateRarity(rebuiltConfig RebuiltConfig, moon string) RebuiltConfig {
	for _, itemConfig := range rebuiltConfig.Items {
		rarity := 0
		count := 0
		for _, moonRarity := range itemConfig.MoonRarity {
			if moon == "all" {
				rarity += moonRarity
				count++
			} else if moon == "moon" {
				rarity += moonRarity
				count++
			}
		}
		itemConfig.ComputedRarity = float32(rarity) / float32(count)
		rebuiltConfig.Items[itemConfig.Name] = itemConfig
	}

	return rebuiltConfig
}

func main() {
	rebuiltConfig := parseAgentConfig()

	rebuiltConfig = parseScrapHardcoded(rebuiltConfig)

	rebuiltConfig = calculateWeightRatios(rebuiltConfig)
	rebuiltConfig = calculateRarity(rebuiltConfig, "all")

	printTables(rebuiltConfig)
}

func printTables(rebuiltConfig RebuiltConfig) {
	items := make([]ItemConfig, 0, len(rebuiltConfig.Items))
	// Convert to array
	for _, itemConfig := range rebuiltConfig.Items {
		items = append(items, itemConfig)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].MinWeightRatio < items[j].MaxWeightRatio
	})

	// Shop Items
	items, filteredOutItems := filterOutShopItems(items)
	fmt.Println("=========")
	fmt.Println("Shop Items")
	fmt.Println("=========")
	printAll(filteredOutItems)
	fmt.Printf("\n\n\n\n")

	// Body Items
	items, filteredOutItems = filterOutBodyItems(items)
	fmt.Println("=========")
	fmt.Println("Body Items")
	fmt.Println("=========")
	printAll(filteredOutItems)
	fmt.Printf("\n\n\n\n")

	// Weightless Items
	items, filteredOutItems = filterOutWeightlessItems(items)
	fmt.Println("=========")
	fmt.Println("Weightless Items")
	fmt.Println("=========")
	printAll(filteredOutItems)
	fmt.Printf("\n\n\n\n")

	// Missing Scrap Values
	items, filteredOutItems = filterOutMissingScrapValues(items)
	fmt.Println("=========")
	fmt.Println("Missing Scrap Values")
	fmt.Println("=========")
	printAll(filteredOutItems)
	fmt.Printf("\n\n\n\n")

	fmt.Println("=========")
	fmt.Println("All Items")
	fmt.Println("=========")
	printAll(items)

}

func filterOutShopItems(items []ItemConfig) (filteredItems []ItemConfig, filteredOutItems []ItemConfig) {
	filteredItems = make([]ItemConfig, 0, len(items))
	filteredOutItems = make([]ItemConfig, 0, len(items))
	for _, itemConfig := range items {
		if !itemConfig.IsShop {
			filteredItems = append(filteredItems, itemConfig)
		} else {
			filteredOutItems = append(filteredOutItems, itemConfig)
		}
	}
	return filteredItems, filteredOutItems
}

func filterOutWeightlessItems(items []ItemConfig) (filteredItems []ItemConfig, filteredOutItems []ItemConfig) {
	filteredItems = make([]ItemConfig, 0, len(items))
	filteredOutItems = make([]ItemConfig, 0, len(items))
	for _, itemConfig := range items {
		if itemConfig.Weight == 0 {
			filteredOutItems = append(filteredOutItems, itemConfig)
		} else {
			filteredItems = append(filteredItems, itemConfig)
		}
	}
	return filteredItems, filteredOutItems
}

func filterOutMissingScrapValues(items []ItemConfig) (filteredItems []ItemConfig, filteredOutItems []ItemConfig) {
	filteredItems = make([]ItemConfig, 0, len(items))
	filteredOutItems = make([]ItemConfig, 0, len(items))
	for _, itemConfig := range items {
		if itemConfig.ScrapValueMin == 0 || itemConfig.ScrapValueMax == 0 {
			filteredOutItems = append(filteredOutItems, itemConfig)
		} else {
			filteredItems = append(filteredItems, itemConfig)
		}
	}
	return filteredItems, filteredOutItems
}

func filterOutBodyItems(items []ItemConfig) (filteredItems []ItemConfig, filteredOutItems []ItemConfig) {
	filteredItems = make([]ItemConfig, 0, len(items))
	filteredOutItems = make([]ItemConfig, 0, len(items))
	for _, itemConfig := range items {
		if itemConfig.IsBody {
			filteredOutItems = append(filteredOutItems, itemConfig)
		} else {
			filteredItems = append(filteredItems, itemConfig)
		}
	}
	return filteredItems, filteredOutItems
}

func printAll(items []ItemConfig) {
	w := tabwriter.NewWriter(os.Stdout, 15, 1, 1, ' ', 0)
	fmt.Fprintln(w, "Name\tScrapMin\tScrapMax\tWeight\tTwoHanded\tCarryEffect\tMetal\tRarity\tMinWeightRatio\tMaxWeightRatio")

	for _, itemConfig := range items {
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f\t%v\t%s\t%v\t%.2f\t%.2f\t%.2f\n",
			itemConfig.Name,
			itemConfig.ScrapValueMin,
			itemConfig.ScrapValueMax,
			itemConfig.Weight,
			itemConfig.TwoHanded,
			itemConfig.CarryEffect,
			itemConfig.IsMetal,
			itemConfig.ComputedRarity,
			itemConfig.MinWeightRatio,
			itemConfig.MaxWeightRatio)
	}

	w.Flush()
}
