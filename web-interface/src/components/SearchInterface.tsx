import { useState, useEffect, useCallback } from "react"
import { Button } from "./ui/button"
import { Input } from "./ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Spinner } from "./ui/spinner"
import { Search, Package, Plus, QrCode, BarChart3, Tag, MapPin } from "lucide-react"
import { api, type PartAPIResponse, type SearchResult } from "../api"

export default function SearchInterface() {
  const [searchQuery, setSearchQuery] = useState("")
  const [searchResults, setSearchResults] = useState<PartAPIResponse[]>([])
  const [allParts, setAllParts] = useState<PartAPIResponse[]>([])
  const [isSearching, setIsSearching] = useState(false)
  const [isLoadingParts, setIsLoadingParts] = useState(false)
  const [searchTimeout, setSearchTimeout] = useState<NodeJS.Timeout | null>(null)

  // Recherche avec debounce
  const performSearch = useCallback(async (query: string) => {
    if (!query.trim()) {
      setSearchResults([])
      return
    }

    setIsSearching(true)
    try {
      const [results, error] = await api.search(query)
      if (error) {
        console.error("Search error:", error)
        setSearchResults([])
      } else {
        setSearchResults(results?.parts || [])
      }
    } catch (error) {
      console.error("Search failed:", error)
      setSearchResults([])
    } finally {
      setIsSearching(false)
    }
  }, [])

  // Gestionnaire de changement de recherche
  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const query = e.target.value
    setSearchQuery(query)

    // Annuler le timeout précédent
    if (searchTimeout) {
      clearTimeout(searchTimeout)
    }

    // Nouveau timeout pour la recherche
    const timeout = setTimeout(() => performSearch(query), 300)
    setSearchTimeout(timeout)
  }

  // Charger toutes les pièces
  const loadAllParts = async () => {
    setIsLoadingParts(true)
    try {
      const [parts, error] = await api.getParts()
      if (error) {
        console.error("Load parts error:", error)
        setAllParts([])
      } else {
        setAllParts(parts || [])
      }
    } catch (error) {
      console.error("Load parts failed:", error)
      setAllParts([])
    } finally {
      setIsLoadingParts(false)
    }
  }

  // Vérifier la santé de l'API au montage
  useEffect(() => {
    const checkHealth = async () => {
      const [healthy, error] = await api.health()
      if (!healthy) {
        console.warn("API health check failed:", error)
      }
    }
    checkHealth()
  }, [])

  // Nettoyer le timeout
  useEffect(() => {
    return () => {
      if (searchTimeout) {
        clearTimeout(searchTimeout)
      }
    }
  }, [searchTimeout])

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted/20">
      <div className="container mx-auto px-4 py-8 max-w-7xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <Package className="h-12 w-12 text-primary" />
            <h1 className="text-4xl font-bold">Open Objects</h1>
          </div>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Gestionnaire intelligent de pièces détachées
          </p>
        </div>

        {/* Actions principales */}
        <div className="flex flex-wrap justify-center gap-4 mb-8">
          <Button asChild size="lg" className="gap-2">
            <a href="/add">
              <Plus className="h-5 w-5" />
              Ajouter une pièce
            </a>
          </Button>
          <Button asChild variant="outline" size="lg" className="gap-2">
            <a href="/scan">
              <QrCode className="h-5 w-5" />
              Scanner QR
            </a>
          </Button>
          <Button
            onClick={loadAllParts}
            disabled={isLoadingParts}
            variant="secondary"
            size="lg"
            className="gap-2"
          >
            {isLoadingParts ? (
              <Spinner size="sm" />
            ) : (
              <Package className="h-5 w-5" />
            )}
            Voir toutes les pièces
          </Button>
        </div>

        {/* Barre de recherche */}
        <div className="max-w-2xl mx-auto mb-8">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Rechercher une pièce (nom, type, référence...)"
              value={searchQuery}
              onChange={handleSearchChange}
              className="pl-12 h-12 text-lg"
            />
            {isSearching && (
              <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
                <Spinner size="sm" />
              </div>
            )}
          </div>
        </div>

        {/* Résultats de recherche */}
        {searchQuery && (
          <Card className="mb-8">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Search className="h-5 w-5" />
                Résultats de recherche
                {searchResults.length > 0 && (
                  <Badge variant="secondary">{searchResults.length}</Badge>
                )}
              </CardTitle>
              {searchQuery && (
                <CardDescription>
                  Recherche pour "{searchQuery}"
                </CardDescription>
              )}
            </CardHeader>
            <CardContent>
              {searchResults.length === 0 && !isSearching ? (
                <div className="text-center py-8">
                  <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <h3 className="text-lg font-semibold mb-2">Aucun résultat</h3>
                  <p className="text-muted-foreground">
                    Essayez avec d'autres termes de recherche
                  </p>
                </div>
              ) : (
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {searchResults.map((part) => (
                    <Card key={part.id} className="hover:shadow-md transition-shadow">
                      <CardHeader className="pb-3">
                        <div className="flex items-start justify-between">
                          <CardTitle className="text-lg">{part.name}</CardTitle>
                          <Badge variant="outline">{part.type}</Badge>
                        </div>
                      </CardHeader>
                      <CardContent className="pt-0">
                        {part.location && (
                          <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
                            <MapPin className="h-4 w-4" />
                            {part.location}
                          </div>
                        )}
                        <div className="text-xs text-muted-foreground">
                          ID: {part.id}
                          {part.source && ` • ${part.source}`}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* Toutes les pièces */}
        {allParts.length > 0 && (
          <Card className="mb-8">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Package className="h-5 w-5" />
                Toutes les pièces
                <Badge variant="secondary">{allParts.length}</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {allParts.map((part) => (
                  <Card key={part.id} className="hover:shadow-md transition-shadow">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <CardTitle className="text-base">{part.name}</CardTitle>
                        <Badge variant="outline" className="text-xs">
                          {part.type}
                        </Badge>
                      </div>
                    </CardHeader>
                    <CardContent className="pt-0">
                      {part.location && (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
                          <MapPin className="h-4 w-4" />
                          {part.location}
                        </div>
                      )}
                      <div className="text-xs text-muted-foreground">
                        ID: {part.id}
                        {part.source && ` • ${part.source}`}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* État initial - pas de recherche, pas de pièces chargées */}
        {!searchQuery && allParts.length === 0 && (
          <div className="text-center py-16">
            <Search className="h-16 w-16 text-muted-foreground mx-auto mb-6" />
            <h2 className="text-2xl font-semibold mb-4">Prêt à rechercher ?</h2>
            <p className="text-muted-foreground text-lg max-w-md mx-auto">
              Utilisez la barre de recherche ci-dessus ou cliquez sur "Voir toutes les pièces"
            </p>
          </div>
        )}

        {/* Statistiques rapides */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-12">
          <Card className="text-center">
            <CardHeader className="pb-3">
              <BarChart3 className="h-8 w-8 text-primary mx-auto mb-2" />
              <CardTitle className="text-lg">Statistiques</CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription>
                Consulter les métriques de votre collection
              </CardDescription>
            </CardContent>
          </Card>

          <Card className="text-center">
            <CardHeader className="pb-3">
              <Tag className="h-8 w-8 text-primary mx-auto mb-2" />
              <CardTitle className="text-lg">Catégories</CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription>
                Organiser vos pièces par type
              </CardDescription>
            </CardContent>
          </Card>

          <Card className="text-center">
            <CardHeader className="pb-3">
              <MapPin className="h-8 w-8 text-primary mx-auto mb-2" />
              <CardTitle className="text-lg">Localisations</CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription>
                Gérer l'emplacement de vos pièces
              </CardDescription>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
