import { useState, useEffect, useRef } from "react"
import { Button } from "./ui/button"
import { Input } from "./ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Plus, X, Upload, MapPin, Camera, Package, Loader2 } from "lucide-react"
import { api } from "../api"

interface TemplateField {
  name: string
  type: string
  label: string
  required?: boolean
  unit?: string
  options?: string[]
}

interface TemplateData {
  fields: TemplateField[]
}
export const prerender = false;
export default function AddPartForm() {
  const [formData, setFormData] = useState({
    type: "",
    name: "",
    location: "",
    locationPath: "",
  })

  const [dynamicFields, setDynamicFields] = useState<Record<string, string>>({})
  const [photos, setPhotos] = useState<File[]>([])
  const [template, setTemplate] = useState<TemplateData | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitResult, setSubmitResult] = useState<{ success?: boolean; message?: string } | null>(null)
  const [locationSuggestions, setLocationSuggestions] = useState<any[]>([])
  const [showLocationSuggestions, setShowLocationSuggestions] = useState(false)
  const [locationSearchTimeout, setLocationSearchTimeout] = useState<NodeJS.Timeout | null>(null)

  const fileInputRef = useRef<HTMLInputElement>(null)

  // Types disponibles
  const availableTypes = [
    { value: "moteur", label: "Moteur" },
    { value: "roulement", label: "Roulement" },
    { value: "vis", label: "Vis" },
    { value: "capteur", label: "Capteur" },
    { value: "resistance", label: "Résistance" },
    { value: "condensateur", label: "Condensateur" },
    { value: "", label: "Autre" },
  ]

  // Charger le template quand le type change
  useEffect(() => {
    if (formData.type && formData.type !== "") {
      loadTemplate(formData.type)
    } else {
      setTemplate(null)
      setDynamicFields({})
    }
  }, [formData.type])

  const loadTemplate = async (type: string) => {
    try {
      const [error, templateData] = await api.getTemplateFields()
      if (error) {
        console.error("Erreur chargement template:", error)
        setTemplate(null)
        return
      }

      // Trouver le template pour ce type
      if (templateData && templateData[type]) {
        setTemplate({ fields: templateData[type] })
      } else {
        setTemplate(null)
      }
    } catch (error) {
      console.error("Erreur chargement template:", error)
      setTemplate(null)
    }
  }

  // Gestionnaire de changement des champs principaux
  const handleFieldChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  // Gestionnaire des champs dynamiques
  const handleDynamicFieldChange = (fieldName: string, value: string) => {
    setDynamicFields(prev => ({ ...prev, [fieldName]: value }))
  }

  // Recherche de localisations
  const handleLocationSearch = (query: string) => {
    if (locationSearchTimeout) {
      clearTimeout(locationSearchTimeout)
    }

    if (query.length < 2) {
      setLocationSuggestions([])
      setShowLocationSuggestions(false)
      return
    }

    const timeout = setTimeout(async () => {
      try {
        const [error, locations] = await api.getLocations()
        if (error) {
          console.error("Erreur recherche localisations:", error)
          return
        }

        // Filtrer les localisations qui correspondent à la recherche
        const filtered = (locations || []).filter((loc: any) =>
          loc.name.toLowerCase().includes(query.toLowerCase()) ||
          (loc.path && loc.path.toLowerCase().includes(query.toLowerCase()))
        )

        setLocationSuggestions(filtered.slice(0, 10)) // Limiter à 10 suggestions
        setShowLocationSuggestions(true)
      } catch (error) {
        console.error("Erreur recherche localisations:", error)
      }
    }, 300)

    setLocationSearchTimeout(timeout)
  }

  // Sélection d'une localisation
  const selectLocation = (location: any) => {
    setFormData(prev => ({
      ...prev,
      location: location.path || location.id.toString(),
      locationPath: location.path || `ID: ${location.id}`
    }))
    setLocationSuggestions([])
    setShowLocationSuggestions(false)
  }

  // Gestionnaire d'ajout de photos
  const handlePhotoAdd = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    const validFiles = files.filter(file =>
      file.type.startsWith('image/') && file.size <= 10 * 1024 * 1024 // 10MB max
    )

    if (validFiles.length + photos.length > 10) {
      alert("Maximum 10 photos autorisées")
      return
    }

    setPhotos(prev => [...prev, ...validFiles])
    // Reset input
    if (fileInputRef.current) {
      fileInputRef.current.value = ""
    }
  }

  // Suppression d'une photo
  const removePhoto = (index: number) => {
    setPhotos(prev => prev.filter((_, i) => i !== index))
  }

  // Soumission du formulaire
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.type || !formData.name) {
      setSubmitResult({ success: false, message: "Le type et le nom sont requis" })
      return
    }

    setIsSubmitting(true)
    setSubmitResult(null)

    try {
      // Préparer les données du formulaire
      const formDataToSend = new FormData()
      formDataToSend.append('type', formData.type)
      formDataToSend.append('name', formData.name)
      formDataToSend.append('loc', formData.location)

      // Ajouter les propriétés dynamiques
      const props: Record<string, any> = {}
      if (template?.fields) {
        template.fields.forEach(field => {
          const value = dynamicFields[field.name]
          if (value) {
            props[field.name] = value
          }
        })
      }
      formDataToSend.append('props', JSON.stringify(props))

      // Ajouter les photos
      photos.forEach((photo, index) => {
        formDataToSend.append(`photo_${index}`, photo)
      })

      // Envoyer à l'API
      const response = await fetch('http://127.0.0.1:8080/api/parts', {
        method: 'POST',
        body: formDataToSend,
      })

      const result = await response.json()

      if (response.ok && result.id) {
        setSubmitResult({
          success: true,
          message: `Pièce ajoutée avec succès ! ID: ${result.id}`
        })

        // Redirection après 2 secondes
        setTimeout(() => {
          window.location.href = `/view?id=${result.id}`
        }, 2000)
      } else {
        throw new Error(result.error || 'Erreur lors de l\'ajout')
      }

    } catch (error) {
      console.error('Erreur soumission:', error)
      setSubmitResult({
        success: false,
        message: error instanceof Error ? error.message : 'Erreur inconnue'
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted/20">
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <Plus className="h-12 w-12 text-primary" />
            <h1 className="text-4xl font-bold">Ajouter une pièce</h1>
          </div>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Ajoutez une nouvelle pièce à votre collection avec photos et propriétés détaillées
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-8">
          {/* Informations de base */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Package className="h-5 w-5" />
                Informations de base
              </CardTitle>
              <CardDescription>
                Les informations essentielles de votre pièce
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Type */}
              <div>
                <label className="block text-sm font-medium mb-2">Type de pièce *</label>
                <select
                  value={formData.type}
                  onChange={(e) => handleFieldChange('type', e.target.value)}
                  className="w-full px-3 py-2 border border-input rounded-md bg-background"
                  required
                >
                  <option value="">Sélectionner un type</option>
                  {availableTypes.map(type => (
                    <option key={type.value} value={type.value}>
                      {type.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Nom */}
              <div>
                <label className="block text-sm font-medium mb-2">Nom de la pièce *</label>
                <Input
                  type="text"
                  value={formData.name}
                  onChange={(e) => handleFieldChange('name', e.target.value)}
                  placeholder="Ex: Moteur 12V, Roulement SKF 6204..."
                  required
                />
              </div>
            </CardContent>
          </Card>

          {/* Propriétés dynamiques */}
          {template?.fields && template.fields.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Propriétés spécifiques</CardTitle>
                <CardDescription>
                  Propriétés spécifiques au type "{availableTypes.find(t => t.value === formData.type)?.label}"
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {template.fields.map(field => (
                  <div key={field.name}>
                    <label className="block text-sm font-medium mb-2">
                      {field.label || field.name}
                      {field.unit && <span className="text-muted-foreground"> ({field.unit})</span>}
                      {field.required && <span className="text-destructive">*</span>}
                    </label>

                    {field.type === 'select' && field.options ? (
                      <select
                        value={dynamicFields[field.name] || ''}
                        onChange={(e) => handleDynamicFieldChange(field.name, e.target.value)}
                        className="w-full px-3 py-2 border border-input rounded-md bg-background"
                        required={field.required}
                      >
                        <option value="">Sélectionner...</option>
                        {field.options.map(option => (
                          <option key={option} value={option}>{option}</option>
                        ))}
                      </select>
                    ) : (
                      <Input
                        type={field.type === 'number' ? 'number' : 'text'}
                        value={dynamicFields[field.name] || ''}
                        onChange={(e) => handleDynamicFieldChange(field.name, e.target.value)}
                        placeholder={field.label || field.name}
                        required={field.required}
                      />
                    )}
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          {/* Localisation */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MapPin className="h-5 w-5" />
                Localisation
              </CardTitle>
              <CardDescription>
                Où se trouve cette pièce dans votre atelier ?
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="relative">
                <label className="block text-sm font-medium mb-2">Rechercher une localisation</label>
                <Input
                  type="text"
                  placeholder="Tapez pour rechercher une localisation..."
                  onChange={(e) => handleLocationSearch(e.target.value)}
                />

                {/* Suggestions */}
                {showLocationSuggestions && locationSuggestions.length > 0 && (
                  <div className="absolute z-10 w-full mt-1 bg-background border border-border rounded-md shadow-lg max-h-60 overflow-y-auto">
                    {locationSuggestions.map((location) => (
                      <div
                        key={location.id}
                        className="px-4 py-2 hover:bg-accent cursor-pointer"
                        onClick={() => selectLocation(location)}
                      >
                        <div className="font-medium">{location.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {location.path || `ID: ${location.id}`}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Localisation sélectionnée */}
              {formData.locationPath && (
                <div className="p-3 bg-accent rounded-md">
                  <div className="flex items-center gap-2">
                    <MapPin className="h-4 w-4 text-primary" />
                    <span className="font-medium">Localisation sélectionnée:</span>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">{formData.locationPath}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Photos */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Camera className="h-5 w-5" />
                Photos
              </CardTitle>
              <CardDescription>
                Ajoutez des photos de votre pièce (max 10, 10MB chacune)
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Bouton d'ajout */}
              <div>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  multiple
                  onChange={handlePhotoAdd}
                  className="hidden"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => fileInputRef.current?.click()}
                  className="gap-2"
                >
                  <Upload className="h-4 w-4" />
                  Ajouter des photos
                </Button>
              </div>

              {/* Aperçu des photos */}
              {photos.length > 0 && (
                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                  {photos.map((photo, index) => (
                    <div key={index} className="relative group">
                      <img
                        src={URL.createObjectURL(photo)}
                        alt={`Photo ${index + 1}`}
                        className="w-full h-24 object-cover rounded-md border"
                      />
                      <button
                        type="button"
                        onClick={() => removePhoto(index)}
                        className="absolute -top-2 -right-2 bg-destructive text-destructive-foreground rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <X className="h-3 w-3" />
                      </button>
                      <div className="absolute bottom-0 left-0 right-0 bg-black/50 text-white text-xs p-1 rounded-b-md">
                        {photo.name}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Résultat de soumission */}
          {submitResult && (
            <Card className={submitResult.success ? "border-green-500" : "border-destructive"}>
              <CardContent className="pt-6">
                <div className={`flex items-center gap-3 ${
                  submitResult.success ? 'text-green-700' : 'text-destructive'
                }`}>
                  {submitResult.success ? (
                    <Package className="h-5 w-5" />
                  ) : (
                    <X className="h-5 w-5" />
                  )}
                  <p className="font-medium">{submitResult.message}</p>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Bouton de soumission */}
          <div className="flex justify-end">
            <Button
              type="submit"
              disabled={isSubmitting}
              size="lg"
              className="gap-2"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Ajout en cours...
                </>
              ) : (
                <>
                  <Plus className="h-4 w-4" />
                  Ajouter la pièce
                </>
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
