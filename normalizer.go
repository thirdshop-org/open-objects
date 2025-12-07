package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// UnitDomain représente un domaine physique (dimension, tension, etc.)
type UnitDomain string

const (
	DomainDimension   UnitDomain = "dimension"   // Longueurs, diamètres -> mm
	DomainTension     UnitDomain = "tension"     // Tension électrique -> V
	DomainCourant     UnitDomain = "courant"     // Intensité -> A
	DomainResistance  UnitDomain = "resistance"  // Résistance -> Ohm
	DomainCapacite    UnitDomain = "capacite"    // Capacité -> uF
	DomainPression    UnitDomain = "pression"    // Pression -> bar
	DomainVitesseRot  UnitDomain = "vitesse_rot" // Vitesse de rotation -> rpm
	DomainPuissance   UnitDomain = "puissance"   // Puissance -> W
	DomainNone        UnitDomain = ""            // Pas de domaine (texte libre)
)

// UnitInfo contient les informations sur une unité
type UnitInfo struct {
	Domain     UnitDomain
	ToBaseFactor float64 // Facteur de conversion vers l'unité de base
}

// BaseUnits définit l'unité de référence pour chaque domaine
var BaseUnits = map[UnitDomain]string{
	DomainDimension:   "mm",
	DomainTension:     "V",
	DomainCourant:     "A",
	DomainResistance:  "Ohm",
	DomainCapacite:    "uF",
	DomainPression:    "bar",
	DomainVitesseRot:  "rpm",
	DomainPuissance:   "W",
}

// UnitConversions mappe les alias d'unités vers leurs informations de conversion
var UnitConversions = map[string]UnitInfo{
	// Dimension (base: mm)
	"mm":    {DomainDimension, 1},
	"cm":    {DomainDimension, 10},
	"m":     {DomainDimension, 1000},
	"in":    {DomainDimension, 25.4},
	"inch":  {DomainDimension, 25.4},
	"pouce": {DomainDimension, 25.4},
	"\"":    {DomainDimension, 25.4},

	// Tension (base: V)
	"v":     {DomainTension, 1},
	"V":     {DomainTension, 1},
	"volt":  {DomainTension, 1},
	"volts": {DomainTension, 1},
	"mV":    {DomainTension, 0.001},
	"mv":    {DomainTension, 0.001},
	"kV":    {DomainTension, 1000},
	"kv":    {DomainTension, 1000},

	// Courant (base: A)
	"a":      {DomainCourant, 1},
	"A":      {DomainCourant, 1},
	"amp":    {DomainCourant, 1},
	"ampere": {DomainCourant, 1},
	"mA":     {DomainCourant, 0.001},
	"ma":     {DomainCourant, 0.001},

	// Résistance (base: Ohm)
	"ohm":   {DomainResistance, 1},
	"Ohm":   {DomainResistance, 1},
	"Ω":     {DomainResistance, 1},
	"kohm":  {DomainResistance, 1000},
	"kOhm":  {DomainResistance, 1000},
	"kΩ":    {DomainResistance, 1000},
	"k":     {DomainResistance, 1000}, // Attention: ambigu mais courant en électronique

	// Capacité (base: uF)
	"F":  {DomainCapacite, 1000000},
	"mF": {DomainCapacite, 1000},
	"uF": {DomainCapacite, 1},
	"µF": {DomainCapacite, 1},
	"nF": {DomainCapacite, 0.001},
	"pF": {DomainCapacite, 0.000001},

	// Pression (base: bar)
	"bar": {DomainPression, 1},
	"psi": {DomainPression, 0.0689476}, // 1 psi ≈ 0.069 bar
	"Pa":  {DomainPression, 0.00001},   // 1 Pa = 0.00001 bar
	"kPa": {DomainPression, 0.01},
	"MPa": {DomainPression, 10},

	// Vitesse de rotation (base: rpm)
	"rpm":    {DomainVitesseRot, 1},
	"tr/min": {DomainVitesseRot, 1},
	"tpm":    {DomainVitesseRot, 1},

	// Puissance (base: W)
	"w":    {DomainPuissance, 1},
	"W":    {DomainPuissance, 1},
	"watt": {DomainPuissance, 1},
	"mW":   {DomainPuissance, 0.001},
	"kW":   {DomainPuissance, 1000},
}

// ParsedValue représente une valeur parsée avec son unité
type ParsedValue struct {
	Value   float64
	Unit    string
	HasUnit bool
}

// parseValueRegex extrait un nombre et son unité optionnelle
var parseValueRegex = regexp.MustCompile(`^([-+]?\d*\.?\d+)\s*([a-zA-ZΩµ"/]+)?$`)

// ParseValueWithUnit parse une chaîne comme "10mm" ou "12.5 cm" ou "10"
func ParseValueWithUnit(input string) (*ParsedValue, error) {
	input = strings.TrimSpace(input)
	
	if input == "" {
		return nil, fmt.Errorf("valeur vide")
	}

	matches := parseValueRegex.FindStringSubmatch(input)
	if matches == nil {
		return nil, fmt.Errorf("format invalide: '%s' (attendu: nombre[unité])", input)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return nil, fmt.Errorf("nombre invalide: '%s'", matches[1])
	}

	unit := strings.TrimSpace(matches[2])

	return &ParsedValue{
		Value:   value,
		Unit:    unit,
		HasUnit: unit != "",
	}, nil
}

// NormalizeResult contient le résultat de la normalisation
type NormalizeResult struct {
	Value    float64    // Valeur normalisée dans l'unité de base
	Domain   UnitDomain // Domaine détecté
	BaseUnit string     // Unité de base utilisée
}

// NormalizeValue convertit une valeur vers l'unité de base du domaine
// Si defaultUnit est fourni, il est utilisé quand aucune unité n'est spécifiée
func NormalizeValue(input string, defaultUnit string) (*NormalizeResult, error) {
	parsed, err := ParseValueWithUnit(input)
	if err != nil {
		return nil, err
	}

	// Déterminer l'unité à utiliser
	unitToUse := parsed.Unit
	if !parsed.HasUnit {
		if defaultUnit == "" {
			// Pas d'unité et pas de défaut -> retourner la valeur telle quelle
			return &NormalizeResult{
				Value:    parsed.Value,
				Domain:   DomainNone,
				BaseUnit: "",
			}, nil
		}
		unitToUse = defaultUnit
	}

	// Chercher l'unité dans les conversions
	info, exists := UnitConversions[unitToUse]
	if !exists {
		// Suggérer les unités valides pour ce type d'entrée
		suggestions := getSuggestionsForUnit(unitToUse)
		if suggestions != "" {
			return nil, fmt.Errorf("unité '%s' non reconnue. %s", unitToUse, suggestions)
		}
		return nil, fmt.Errorf("unité '%s' non reconnue", unitToUse)
	}

	// Convertir vers l'unité de base
	normalizedValue := parsed.Value * info.ToBaseFactor

	return &NormalizeResult{
		Value:    normalizedValue,
		Domain:   info.Domain,
		BaseUnit: BaseUnits[info.Domain],
	}, nil
}

// getSuggestionsForUnit retourne des suggestions basées sur le domaine probable
func getSuggestionsForUnit(unit string) string {
	lower := strings.ToLower(unit)
	
	// Essayer de deviner le domaine basé sur des patterns communs
	suggestions := map[string][]string{
		"dimension":   {"mm", "cm", "m", "in", "inch", "pouce"},
		"tension":     {"V", "mV", "kV", "volt"},
		"courant":     {"A", "mA", "amp"},
		"resistance":  {"Ohm", "kOhm", "Ω"},
		"capacite":    {"uF", "µF", "nF", "pF", "mF", "F"},
		"pression":    {"bar", "psi", "Pa", "kPa"},
		"vitesse_rot": {"rpm", "tr/min", "tpm"},
		"puissance":   {"W", "kW", "mW", "watt"},
	}

	// Patterns pour deviner le domaine
	if strings.Contains(lower, "m") || strings.Contains(lower, "inch") || strings.Contains(lower, "pouce") {
		return "Unités de dimension valides: " + strings.Join(suggestions["dimension"], ", ")
	}
	if strings.Contains(lower, "v") || strings.Contains(lower, "volt") {
		return "Unités de tension valides: " + strings.Join(suggestions["tension"], ", ")
	}
	if strings.Contains(lower, "a") || strings.Contains(lower, "amp") {
		return "Unités de courant valides: " + strings.Join(suggestions["courant"], ", ")
	}
	if strings.Contains(lower, "ohm") || strings.Contains(lower, "ω") {
		return "Unités de résistance valides: " + strings.Join(suggestions["resistance"], ", ")
	}
	if strings.Contains(lower, "f") {
		return "Unités de capacité valides: " + strings.Join(suggestions["capacite"], ", ")
	}
	if strings.Contains(lower, "bar") || strings.Contains(lower, "psi") || strings.Contains(lower, "pa") {
		return "Unités de pression valides: " + strings.Join(suggestions["pression"], ", ")
	}
	if strings.Contains(lower, "rpm") || strings.Contains(lower, "tr") || strings.Contains(lower, "min") {
		return "Unités de vitesse valides: " + strings.Join(suggestions["vitesse_rot"], ", ")
	}
	if strings.Contains(lower, "w") || strings.Contains(lower, "watt") {
		return "Unités de puissance valides: " + strings.Join(suggestions["puissance"], ", ")
	}

	return ""
}

// GetDefaultUnitForField retourne l'unité par défaut pour un champ donné
// Basé sur des conventions de nommage courantes
func GetDefaultUnitForField(fieldName string) string {
	lower := strings.ToLower(fieldName)
	
	// Champs de dimension
	dimensionFields := []string{
		"d_int", "d_ext", "diametre", "diameter", "largeur", "longueur",
		"length", "width", "height", "hauteur", "epaisseur", "thickness",
		"axe", "rayon", "radius", "taille", "size", "pas",
	}
	for _, f := range dimensionFields {
		if lower == f || strings.Contains(lower, f) {
			return "mm"
		}
	}

	// Champs de tension
	tensionFields := []string{"volt", "volts", "tension", "voltage"}
	for _, f := range tensionFields {
		if lower == f || strings.Contains(lower, f) {
			return "V"
		}
	}

	// Champs de courant
	courantFields := []string{"amp", "ampere", "courant", "current", "intensite"}
	for _, f := range courantFields {
		if lower == f || strings.Contains(lower, f) {
			return "A"
		}
	}

	// Champs de résistance
	resistanceFields := []string{"ohm", "resistance", "impedance"}
	for _, f := range resistanceFields {
		if lower == f || strings.Contains(lower, f) {
			return "Ohm"
		}
	}

	// Champs de capacité
	capaciteFields := []string{"capacite", "capacity", "farad", "capa"}
	for _, f := range capaciteFields {
		if lower == f || strings.Contains(lower, f) {
			return "uF"
		}
	}

	// Champs de pression
	pressionFields := []string{"pression", "pressure"}
	for _, f := range pressionFields {
		if lower == f || strings.Contains(lower, f) {
			return "bar"
		}
	}

	// Champs de vitesse de rotation
	vitesseFields := []string{"rpm", "vitesse", "speed", "rotation", "tr/min", "tours"}
	for _, f := range vitesseFields {
		if lower == f || strings.Contains(lower, f) {
			return "rpm"
		}
	}

	// Champs de puissance
	puissanceFields := []string{"watt", "watts", "puissance", "power"}
	for _, f := range puissanceFields {
		if lower == f || strings.Contains(lower, f) {
			return "W"
		}
	}

	return "" // Pas d'unité par défaut
}

// textOnlyFields liste les champs qui sont toujours du texte libre (jamais des valeurs avec unités)
var textOnlyFields = map[string]bool{
	"reference": true,
	"marque":    true,
	"type":      true,
	"tete":      true,
	"materiau":  true,
	"modele":    true,
	"serie":     true,
	"nom":       true,
	"name":      true,
	"couleur":   true,
	"notes":     true,
	"commentaire": true,
}

// isTextOnlyField vérifie si un champ est toujours du texte
func isTextOnlyField(fieldName string) bool {
	return textOnlyFields[strings.ToLower(fieldName)]
}

// NormalizeProps normalise toutes les propriétés numériques d'un map
// Les valeurs peuvent être des nombres, des chaînes avec unités, ou du texte libre
func NormalizeProps(props map[string]interface{}, fieldUnits map[string]string) (map[string]interface{}, error) {
	normalized := make(map[string]interface{})

	for key, value := range props {
		// Champs texte: ne pas essayer de normaliser
		if isTextOnlyField(key) {
			normalized[key] = value
			continue
		}

		// Déterminer l'unité par défaut pour ce champ
		defaultUnit := ""
		if fieldUnits != nil {
			if u, ok := fieldUnits[key]; ok {
				defaultUnit = u
			}
		}
		if defaultUnit == "" {
			defaultUnit = GetDefaultUnitForField(key)
		}

		// Traiter selon le type de valeur
		switch v := value.(type) {
		case float64:
			// Nombre pur: appliquer l'unité par défaut si elle existe
			if defaultUnit != "" {
				result, err := NormalizeValue(fmt.Sprintf("%g%s", v, defaultUnit), "")
				if err != nil {
					return nil, fmt.Errorf("champ '%s': %v", key, err)
				}
				normalized[key] = result.Value
			} else {
				normalized[key] = v
			}

		case int:
			// Entier: même logique
			if defaultUnit != "" {
				result, err := NormalizeValue(fmt.Sprintf("%d%s", v, defaultUnit), "")
				if err != nil {
					return nil, fmt.Errorf("champ '%s': %v", key, err)
				}
				normalized[key] = result.Value
			} else {
				normalized[key] = float64(v)
			}

		case string:
			// Chaîne: essayer de parser comme valeur + unité
			_, parseErr := ParseValueWithUnit(v)
			if parseErr != nil {
				// Pas un nombre, garder comme texte
				normalized[key] = v
				continue
			}

			// C'est une valeur numérique, normaliser
			result, err := NormalizeValue(v, defaultUnit)
			if err != nil {
				return nil, fmt.Errorf("champ '%s': %v", key, err)
			}
			normalized[key] = result.Value

		default:
			// Autre type: garder tel quel
			normalized[key] = value
		}
	}

	return normalized, nil
}

// GetAcceptedUnitsForDomain retourne les unités acceptées pour un domaine
func GetAcceptedUnitsForDomain(domain UnitDomain) []string {
	var units []string
	for unit, info := range UnitConversions {
		if info.Domain == domain {
			units = append(units, unit)
		}
	}
	return units
}

// ValidateUnitForField vérifie qu'une unité est valide pour un champ donné
func ValidateUnitForField(fieldName string, unit string) error {
	if unit == "" {
		return nil
	}

	info, exists := UnitConversions[unit]
	if !exists {
		return fmt.Errorf("unité '%s' non reconnue", unit)
	}

	// Vérifier la cohérence avec le champ
	expectedUnit := GetDefaultUnitForField(fieldName)
	if expectedUnit == "" {
		return nil // Champ sans domaine spécifique
	}

	expectedInfo, _ := UnitConversions[expectedUnit]
	if info.Domain != expectedInfo.Domain {
		return fmt.Errorf("unité '%s' incompatible avec le champ '%s' (attendu: %s)", 
			unit, fieldName, BaseUnits[expectedInfo.Domain])
	}

	return nil
}

