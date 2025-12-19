import { useState, useEffect, useRef } from "react"
import { Button } from "./ui/button"
import { Input } from "./ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Plus, X, Upload, MapPin, Camera, Package, Loader2, ChevronLeft, Check } from "lucide-react"
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

export const prerender = false

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
  const [availableTypes, setAvailableTypes] = useState<{ value: string; label: string; description?: string }[]>([])
  const [typesLoading, setTypesLoading] = useState(true)

  const fileInputRef = useRef<HTMLInputElement>(null)
  const cameraInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const loadAvailableTypes = async () => {
      try {
        const [error, types] = await api.getPartTypes()
        if (error) {
          console.error("Erreur chargement types:", error)
          setAvailableTypes([
            { value: "", label: "Autre" },
            { value: "moteur", label: "Moteur" },
            { value: "roulement", label: "Roulement" },
            { value: "vis", label: "Vis" },
          ])
        } else {
          setAvailableTypes(types || [])
        }
      } catch (error) {
        console.error("Erreur chargement types:", error)
        setAvailableTypes([
          { value: "", label: "Autre" },
          { value: "moteur", label: "Moteur" },
          { value: "roulement", label: "Roulement" },
          { value: "vis", label: "Vis" },
        ])
      } finally {
        setTypesLoading(false)
      }
    }

    loadAvailableTypes()
  }, [])

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
      const [error, templateData] = await api.getTemplateFields(type)
      if (error) {
        console.error("Erreur chargement template:", error)
        setTemplate(null)
        return
      }

      if (templateData && templateData.fields) {
        setTemplate({ fields: templateData.fields })
      } else {
        setTemplate(null)
      }
    } catch (error) {
      console.error("Erreur chargement template:", error)
      setTemplate(null)
    }
  }

  const handleFieldChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const handleDynamicFieldChange = (fieldName: string, value: string) => {
    setDynamicFields(prev => ({ ...prev, [fieldName]: value }))
  }

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
        const [error, locations] = await api.getLocations({
          search: query,
          limit: 10
        })

        if (error) {
          console.error("Erreur recherche localisations:", error)
          return
        }

        setLocationSuggestions(locations || [])
        setShowLocationSuggestions(true)
      } catch (error) {
        console.error("Erreur recherche localisations:", error)
      }
    }, 300)

    setLocationSearchTimeout(timeout)
  }

  const selectLocation = (location: any) => {
    setFormData(prev => ({
      ...prev,
      location: location.path || location.id.toString(),
      locationPath: location.path || `ID: ${location.id}`
    }))
    setLocationSuggestions([])
    setShowLocationSuggestions(false)
  }

  const handlePhotoAdd = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    const validFiles = files.filter(file =>
      file.type.startsWith('image/') && file.size <= 10 * 1024 * 1024
    )

    if (validFiles.length + photos.length > 10) {
      alert("Maximum 10 photos autorisées")
      return
    }

    setPhotos(prev => [...prev, ...validFiles])
    if (fileInputRef.current) {
      fileInputRef.current.value = ""
    }
    if (cameraInputRef.current) {
      cameraInputRef.current.value = ""
    }
  }

  const removePhoto = (index: number) => {
    setPhotos(prev => prev.filter((_, i) => i !== index))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.type || !formData.name) {
      setSubmitResult({ success: false, message: "Le type et le nom sont requis" })
      return
    }

    setIsSubmitting(true)
    setSubmitResult(null)

    try {
      const props: Record<string, any> = {}
      if (template?.fields) {
        template.fields.forEach(field => {
          const value = dynamicFields[field.name]
          if (value) {
            props[field.name] = value
          }
        })
      }

      const partData = {
        type: formData.type || undefined,
        name: formData.name,
        loc: formData.location || undefined,
        props: Object.keys(props).length > 0 ? props : undefined,
      }

      const [error, result] = await api.addPart(partData, photos.length > 0 ? photos : undefined)

      if (error) {
        throw new Error(error)
      }

      if (result && result.id) {
        setSubmitResult({
          success: true,
          message: `Pièce ajoutée avec succès !`
        })

        setTimeout(() => {
          window.location.href = `/view?id=${result.id}`
        }, 2000)
      } else {
        throw new Error(result?.error || 'Erreur lors de l\'ajout')
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
    <div className="min-h-screen bg-background pb-24">
      {/* Header fixe */}
      <div className="sticky top-0 z-40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 border-b">
        <div className="flex items-center gap-3 px-4 py-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => window.history.back()}
            className="shrink-0"
          >
            <ChevronLeft className="h-6 w-6" />
          </Button>
          <div className="flex items-center gap-2 min-w-0">
            <Plus className="h-6 w-6 text-primary shrink-0" />
            <h1 className="text-xl font-bold truncate">Ajouter une pièce</h1>
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="px-4 py-6 space-y-6">
        {/* Informations de base */}
        <div className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Package className="h-5 w-5" />
            Informations de base
          </h2>

          {/* Type */}
          <div>
            <label className="block text-sm font-medium mb-2">
              Type de pièce <span className="text-destructive">*</span>
            </label>
            <select
              value={formData.type}
              onChange={(e) => handleFieldChange('type', e.target.value)}
              className="w-full h-12 px-4 border border-input rounded-lg bg-background text-base"
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
            <label className="block text-sm font-medium mb-2">
              Nom de la pièce <span className="text-destructive">*</span>
            </label>
            <Input
              type="text"
              value={formData.name}
              onChange={(e) => handleFieldChange('name', e.target.value)}
              placeholder="Ex: Moteur 12V, Roulement SKF..."
              className="h-12 text-base"
              required
            />
          </div>
        </div>

        {/* Propriétés dynamiques */}
        {template?.fields && template.fields.length > 0 && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold">Propriétés spécifiques</h2>
            <p className="text-sm text-muted-foreground">
              Pour "{availableTypes.find(t => t.value === formData.type)?.label}"
            </p>

            {template.fields.map(field => (
              <div key={field.name}>
                <label className="block text-sm font-medium mb-2">
                  {field.label || field.name}
                  {field.unit && <span className="text-muted-foreground"> ({field.unit})</span>}
                  {field.required && <span className="text-destructive"> *</span>}
                </label>

                {field.type === 'select' && field.options ? (
                  <select
                    value={dynamicFields[field.name] || ''}
                    onChange={(e) => handleDynamicFieldChange(field.name, e.target.value)}
                    className="w-full h-12 px-4 border border-input rounded-lg bg-background text-base"
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
                    className="h-12 text-base"
                    required={field.required}
                  />
                )}
              </div>
            ))}
          </div>
        )}

        {/* Localisation */}
        <div className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <MapPin className="h-5 w-5" />
            Localisation
          </h2>

          <div className="relative">
            <Input
              type="text"
              placeholder="Rechercher une localisation..."
              onChange={(e) => handleLocationSearch(e.target.value)}
              className="h-12 text-base"
            />

            {/* Suggestions */}
            {showLocationSuggestions && locationSuggestions.length > 0 && (
              <div className="absolute z-50 w-full mt-2 bg-background border border-border rounded-lg shadow-lg max-h-64 overflow-y-auto">
                {locationSuggestions.map((location) => (
                  <button
                    type="button"
                    key={location.id}
                    className="w-full px-4 py-3 text-left hover:bg-accent active:bg-accent/80 border-b last:border-b-0"
                    onClick={() => selectLocation(location)}
                  >
                    <div className="font-medium">{location.name}</div>
                    <div className="text-sm text-muted-foreground">
                      {location.path || `ID: ${location.id}`}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Localisation sélectionnée */}
          {formData.locationPath && (
            <div className="p-4 bg-accent rounded-lg">
              <div className="flex items-center gap-2 text-sm font-medium mb-1">
                <Check className="h-4 w-4 text-primary" />
                Localisation sélectionnée
              </div>
              <p className="text-sm text-muted-foreground">{formData.locationPath}</p>
            </div>
          )}
        </div>

        {/* Photos */}
        <div className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Camera className="h-5 w-5" />
            Photos {photos.length > 0 && `(${photos.length}/10)`}
          </h2>

          {/* Boutons d'ajout photo */}
          <div className="grid grid-cols-2 gap-3">
            <input
              ref={cameraInputRef}
              type="file"
              accept="image/*"
              capture="environment"
              multiple
              onChange={handlePhotoAdd}
              className="hidden"
            />
            <Button
              type="button"
              variant="outline"
              onClick={() => cameraInputRef.current?.click()}
              className="h-16 flex-col gap-1"
            >
              <Camera className="h-6 w-6" />
              <span className="text-xs">Prendre photo</span>
            </Button>

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
              className="h-16 flex-col gap-1"
            >
              <Upload className="h-6 w-6" />
              <span className="text-xs">Galerie</span>
            </Button>
          </div>

          {/* Aperçu des photos */}
          {photos.length > 0 && (
            <div className="grid grid-cols-3 gap-2">
              {photos.map((photo, index) => (
                <div key={index} className="relative aspect-square">
                  <img
                    src={URL.createObjectURL(photo)}
                    alt={`Photo ${index + 1}`}
                    className="w-full h-full object-cover rounded-lg border"
                  />
                  <button
                    type="button"
                    onClick={() => removePhoto(index)}
                    className="absolute -top-2 -right-2 bg-destructive text-destructive-foreground rounded-full p-1.5 shadow-md"
                  >
                    <X className="h-3 w-3" />
                  </button>
                  <div className="absolute bottom-0 left-0 right-0 bg-black/60 text-white text-xs p-1 text-center rounded-b-lg truncate">
                    {index + 1}
                  </div>
                </div>
              ))}
            </div>
          )}

          {photos.length === 0 && (
            <div className="text-center py-8 text-muted-foreground text-sm">
              Aucune photo ajoutée
            </div>
          )}
        </div>

        {/* Résultat de soumission */}
        {submitResult && (
          <div className={`p-4 rounded-lg border-2 ${
            submitResult.success
              ? 'bg-green-50 border-green-500 text-green-700'
              : 'bg-red-50 border-red-500 text-red-700'
          }`}>
            <div className="flex items-center gap-3">
              {submitResult.success ? (
                <Check className="h-5 w-5 shrink-0" />
              ) : (
                <X className="h-5 w-5 shrink-0" />
              )}
              <p className="font-medium text-sm">{submitResult.message}</p>
            </div>
          </div>
        )}
      </form>

      {/* Bouton de soumission fixe en bas */}
      <div className="fixed bottom-0 left-0 right-0 bg-background border-t p-4 safe-area-inset-bottom">
        <Button
          type="submit"
          disabled={isSubmitting}
          size="lg"
          className="w-full h-14 text-base gap-2"
          onClick={handleSubmit}
        >
          {isSubmitting ? (
            <>
              <Loader2 className="h-5 w-5 animate-spin" />
              Ajout en cours...
            </>
          ) : (
            <>
              <Plus className="h-5 w-5" />
              Ajouter la pièce
            </>
          )}
        </Button>
      </div>
    </div>
  )
}
