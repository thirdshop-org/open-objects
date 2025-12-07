package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SearchCriteria représente un critère de recherche sur une propriété
type SearchCriteria struct {
	PropName string
	IsRange  bool
	ExactVal string
	MinVal   float64
	MaxVal   float64
}

// ParseSearchProp parse une prop comme "d_int:10..10.5" ou "type:billes"
func ParseSearchProp(prop string) (*SearchCriteria, error) {
	parts := strings.SplitN(prop, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("format invalide: %s (attendu: prop:valeur ou prop:min..max)", prop)
	}

	criteria := &SearchCriteria{PropName: parts[0]}
	value := parts[1]

	// Vérifier si c'est un range (contient ..)
	rangeRegex := regexp.MustCompile(`^([\d.]+)\.\.([\d.]+)$`)
	if matches := rangeRegex.FindStringSubmatch(value); matches != nil {
		min, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return nil, fmt.Errorf("valeur min invalide: %s", matches[1])
		}
		max, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return nil, fmt.Errorf("valeur max invalide: %s", matches[2])
		}
		criteria.IsRange = true
		criteria.MinVal = min
		criteria.MaxVal = max
	} else {
		criteria.IsRange = false
		criteria.ExactVal = value
	}

	return criteria, nil
}

// MatchesCriteria vérifie si une valeur correspond au critère
func (c *SearchCriteria) MatchesCriteria(propVal interface{}) bool {
	if c.IsRange {
		numVal, ok := toFloat64(propVal)
		if !ok {
			return false
		}
		return numVal >= c.MinVal && numVal <= c.MaxVal
	}

	// Recherche exacte
	propStr := fmt.Sprintf("%v", propVal)
	return propStr == c.ExactVal
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

