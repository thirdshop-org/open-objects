import { useState, useEffect } from "react"
import { Button } from "./ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Spinner } from "./ui/spinner"
import { ArrowLeft, Package, MapPin, Settings, Move, AlertCircle } from "lucide-react"
import { api, type PartAPIResponse } from "../api"

export default function ViewPartDetail() {
  const [part, setPart] = useState<PartAPIResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    // Récupérer l'ID depuis les query parameters (format: /view?id=123)
    const urlParams = new URLSearchParams(window.location.search)
    const idStr = urlParams.get('id')

    if (!idStr) {
      setError("ID de pièce manquant (paramètre 'id' requis)")
      setLoading(false)
      return
    }

    const id = parseInt(idStr)

    if (isNaN(id) || id <= 0) {
      setError("ID de pièce invalide")
      setLoading(false)
      return
    }

    loadPart(id)
  }, [])

  const loadPart = async (id: number) => {
    try {
      setLoading(true)
      setError(null)

      // Récupérer les données depuis l'endpoint HTML existant
      const response = await fetch(`http://127.0.0.1:8080/view/${id}`)
      if (!response.ok) {
        throw new Error(`Pièce non trouvée (HTTP ${response.status})`)
      }

      const html = await response.text()
      const parsedPart = parsePartFromHTML(html)

      if (!parsedPart || !parsedPart.id) {
        throw new Error("Impossible d'analyser les données de la pièce")
      }

      setPart(parsedPart)
    } catch (err) {
      console.error("Erreur chargement pièce:", err)
      setError(err instanceof Error ? err.message : "Erreur inconnue")
    } finally {
      setLoading(false)
    }
  }

  // Fonction pour analyser le HTML (solution temporaire)
  const parsePartFromHTML = (html: string): PartAPIResponse | null => {
    try {
      const nameMatch = html.match(/<div class="title">([^<]+).*?\(#(\d+)\)/)
      const typeMatch = html.match(/Type : ([^<]+)/)
      const locationMatch = html.match(/📍 ([^<]+)/)
      const propsMatch = html.match(/<pre>([\s\S]*?)<\/pre>/)

      if (!nameMatch || !typeMatch) {
        return null
      }

      let props = {}
      if (propsMatch) {
        try {
          props = JSON.parse(propsMatch[1].trim())
        } catch (e) {
          console.warn("Impossible de parser les propriétés JSON:", e)
        }
      }

      return {
        id: parseInt(nameMatch[2]),
        name: nameMatch[1].trim(),
        type: typeMatch[1].trim(),
        location: locationMatch ? locationMatch[1].trim() : undefined,
        props: props,
      }
    } catch (err) {
      console.error("Erreur parsing HTML:", err)
      return null
    }
  }

  const formatProps = (props: any): string => {
    return JSON.stringify(props, null, 2)
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background to-muted/20">
        <div className="container mx-auto px-4 py-8 max-w-4xl">
          <div className="flex items-center justify-center min-h-[400px]">
            <div className="text-center">
              <Spinner size="lg" className="mx-auto mb-4" />
              <p className="text-muted-foreground">Chargement de la pièce...</p>
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (error || !part) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background to-muted/20">
        <div className="container mx-auto px-4 py-8 max-w-4xl">
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <div className="text-center">
                <AlertCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
                <h2 className="text-xl font-semibold mb-2">Erreur</h2>
                <p className="text-muted-foreground mb-4">
                  {error || "Pièce non trouvée"}
                </p>
                <Button onClick={() => window.history.back()}>
                  <ArrowLeft className="h-4 w-4 mr-2" />
                  Retour
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted/20">
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        {/* Header avec navigation */}
        <div className="mb-6">
          <Button
            variant="ghost"
            onClick={() => window.history.back()}
            className="mb-4 gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Retour
          </Button>
        </div>

        {/* Informations principales */}
        <Card className="mb-6">
          <CardHeader>
            <div className="flex items-start justify-between">
              <div>
                <CardTitle className="text-3xl flex items-center gap-3">
                  <Package className="h-8 w-8" />
                  {part.name}
                </CardTitle>
                <CardDescription className="text-lg mt-2">
                  ID: {part.id}
                </CardDescription>
              </div>
              <Badge variant="outline" className="text-lg px-3 py-1">
                {part.type}
              </Badge>
            </div>
          </CardHeader>
        </Card>

        {/* Localisation */}
        {part.location && (
          <Card className="mb-6">
            <CardContent className="pt-6">
              <div className="flex items-center gap-3">
                <MapPin className="h-5 w-5 text-primary" />
                <div>
                  <h3 className="font-semibold">Localisation</h3>
                  <p className="text-muted-foreground">{part.location}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Propriétés */}
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Propriétés
            </CardTitle>
            <CardDescription>
              Caractéristiques techniques de la pièce
            </CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="bg-muted p-4 rounded-lg overflow-auto text-sm">
              {formatProps(part.props)}
            </pre>
          </CardContent>
        </Card>

        {/* Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Actions</CardTitle>
            <CardDescription>
              Gérer cette pièce dans votre collection
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-3">
              <Button disabled variant="outline" className="gap-2">
                <Package className="h-4 w-4" />
                Sortir du stock
                <span className="text-xs opacity-60">(API à implémenter)</span>
              </Button>

              <Button disabled variant="outline" className="gap-2">
                <Move className="h-4 w-4" />
                Déplacer
                <span className="text-xs opacity-60">(API à implémenter)</span>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
