import { useState, useEffect, useCallback } from "react"
import { Button } from "./ui/button"
import { Input } from "./ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Spinner } from "./ui/spinner"
import { Search, Package, Plus, QrCode, MapPin, ChevronRight, X } from "lucide-react"
import { api, type PartAPIResponse } from "../api"

export default function SearchInterface() {
  const [searchQuery, setSearchQuery] = useState("")
  const [searchResults, setSearchResults] = useState<PartAPIResponse[]>([])
  const [allParts, setAllParts] = useState<PartAPIResponse[]>([])
  const [isSearching, setIsSearching] = useState(false)
  const [isLoadingParts, setIsLoadingParts] = useState(false)
  const [searchTimeout, setSearchTimeout] = useState<NodeJS.Timeout | null>(null)
  const [selectedPart, setSelectedPart] = useState<PartAPIResponse | null>(null)

  // Recherche avec debounce
  const performSearch = useCallback(async (query: string) => {
    if (!query.trim()) {
      setSearchResults([])
      return
    }

    setIsSearching(true)
    try {
      const [error, results] = await api.search(query)
      if (error) {
        console.error("Search error:", error)
        setSearchResults([])
      } else {
        setSearchResults(results || [])
      }
    } catch (error) {
      console.error("Search failed:", error)
      setSearchResults([])
    } finally {
      setIsSearching(false)
    }
  }, [])

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const query = e.target.value
    setSearchQuery(query)

    if (searchTimeout) {
      clearTimeout(searchTimeout)
    }

    const timeout = setTimeout(() => performSearch(query), 300)
    setSearchTimeout(timeout)
  }

  const clearSearch = () => {
    setSearchQuery("")
    setSearchResults([])
  }

  const loadAllParts = async () => {
    setIsLoadingParts(true)
    try {
      const [error, parts] = await api.getParts()
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

  useEffect(() => {
    const checkHealth = async () => {
      const [error, healthy] = await api.health()
      if (error) {
        console.warn("API health check failed:", error)
      }
    }
    checkHealth()
  }, [])

  useEffect(() => {
    return () => {
      if (searchTimeout) {
        clearTimeout(searchTimeout)
      }
    }
  }, [searchTimeout])

  // Composant de carte de pièce optimisé mobile
  const PartCard = ({ part, onClick }: { part: PartAPIResponse; onClick: () => void }) => (
    <button
      onClick={onClick}
      className="w-full text-left bg-card hover:bg-accent/50 active:bg-accent transition-colors rounded-lg border p-4 flex items-center gap-3"
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2 mb-1">
          <h3 className="font-semibold text-base truncate">{part.name}</h3>
          <Badge variant="outline" className="text-xs shrink-0">
            {part.type}
          </Badge>
        </div>
        {part.location && (
          <div className="flex items-center gap-1.5 text-sm text-muted-foreground mb-1">
            <MapPin className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">{part.location}</span>
          </div>
        )}
        <div className="text-xs text-muted-foreground truncate">
          ID: {part.id}
        </div>
      </div>
      <ChevronRight className="h-5 w-5 text-muted-foreground shrink-0" />
    </button>
  )

  // Modal de détails (optimisé mobile)
  const PartDetailsModal = ({ part, onClose }: { part: PartAPIResponse; onClose: () => void }) => (
    <div className="fixed inset-0 bg-background/95 z-50 overflow-y-auto">
      <div className="min-h-screen p-4">
        <div className="flex items-center justify-between mb-6 sticky top-0 bg-background py-4">
          <h2 className="text-xl font-bold">Détails de la pièce</h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-6 w-6" />
          </Button>
        </div>
        
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{part.name}</CardTitle>
              <CardDescription>
                <Badge variant="outline">{part.type}</Badge>
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {part.location && (
                <div>
                  <div className="text-sm font-medium mb-1">Localisation</div>
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <MapPin className="h-4 w-4" />
                    {part.location}
                  </div>
                </div>
              )}
              <div>
                <div className="text-sm font-medium mb-1">Identifiant</div>
                <div className="text-muted-foreground font-mono text-sm">{part.id}</div>
              </div>
              {part.source && (
                <div>
                  <div className="text-sm font-medium mb-1">Source</div>
                  <div className="text-muted-foreground">{part.source}</div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen bg-background pb-20">
      {/* Header fixe */}
      <div className="sticky top-0 z-40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 border-b">
        <div className="px-4 py-4">
          <div className="flex items-center justify-center gap-2 mb-4">
            <Package className="h-8 w-8 text-primary" />
            <h1 className="text-2xl font-bold">Open Objects</h1>
          </div>
          
          {/* Barre de recherche */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Rechercher..."
              value={searchQuery}
              onChange={handleSearchChange}
              className="pl-10 pr-10 h-12 text-base"
            />
            {searchQuery && (
              <button
                onClick={clearSearch}
                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground"
              >
                <X className="h-5 w-5" />
              </button>
            )}
            {isSearching && (
              <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
                <Spinner size="sm" />
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="px-4 py-4">
        {/* Actions rapides - design mobile-first */}
        {!searchQuery && allParts.length === 0 && (
          <div className="grid grid-cols-2 gap-3 mb-6">
            <Button asChild size="lg" className="h-24 flex-col gap-2">
              <a href="/add">
                <Plus className="h-8 w-8" />
                <span className="text-sm">Ajouter</span>
              </a>
            </Button>
            <Button asChild variant="outline" size="lg" className="h-24 flex-col gap-2">
              <a href="/scan">
                <QrCode className="h-8 w-8" />
                <span className="text-sm">Scanner</span>
              </a>
            </Button>
          </div>
        )}

        {!searchQuery && allParts.length === 0 && (
          <Button
            onClick={loadAllParts}
            disabled={isLoadingParts}
            variant="secondary"
            size="lg"
            className="w-full h-16 gap-2 text-base mb-6"
          >
            {isLoadingParts ? (
              <Spinner size="sm" />
            ) : (
              <Package className="h-6 w-6" />
            )}
            Voir toutes les pièces
          </Button>
        )}

        {/* Résultats de recherche */}
        {searchQuery && (
          <div className="mb-6">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-lg font-semibold">
                Résultats {searchResults.length > 0 && `(${searchResults.length})`}
              </h2>
            </div>
            
            {searchResults.length === 0 && !isSearching ? (
              <div className="text-center py-12">
                <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <p className="text-muted-foreground">Aucun résultat trouvé</p>
              </div>
            ) : (
              <div className="space-y-2">
                {searchResults.map((part) => (
                  <PartCard
                    key={part.id}
                    part={part}
                    onClick={() => setSelectedPart(part)}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Toutes les pièces */}
        {allParts.length > 0 && (
          <div>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-lg font-semibold">
                Mes pièces ({allParts.length})
              </h2>
              <Button
                onClick={() => setAllParts([])}
                variant="ghost"
                size="sm"
              >
                Masquer
              </Button>
            </div>
            
            <div className="space-y-2">
              {allParts.map((part) => (
                <PartCard
                  key={part.id}
                  part={part}
                  onClick={() => setSelectedPart(part)}
                />
              ))}
            </div>
          </div>
        )}

        {/* État initial */}
        {!searchQuery && allParts.length === 0 && (
          <div className="text-center py-8">
            <Package className="h-16 w-16 text-muted-foreground mx-auto mb-4" />
            <h2 className="text-xl font-semibold mb-2">Bienvenue !</h2>
            <p className="text-muted-foreground px-4">
              Commencez par ajouter une pièce ou recherchez dans votre inventaire
            </p>
          </div>
        )}
      </div>

      {/* Modal de détails */}
      {selectedPart && (
        <PartDetailsModal
          part={selectedPart}
          onClose={() => setSelectedPart(null)}
        />
      )}

      {/* Barre de navigation fixe en bas */}
      <div className="fixed bottom-0 left-0 right-0 bg-background border-t safe-area-inset-bottom">
        <div className="grid grid-cols-3 gap-1 p-2">
          <Button asChild variant="ghost" className="flex-col h-16 gap-1">
            <a href="/add">
              <Plus className="h-5 w-5" />
              <span className="text-xs">Ajouter</span>
            </a>
          </Button>
          <Button asChild variant="ghost" className="flex-col h-16 gap-1">
            <a href="/scan">
              <QrCode className="h-5 w-5" />
              <span className="text-xs">Scanner</span>
            </a>
          </Button>
          <Button
            onClick={loadAllParts}
            disabled={isLoadingParts}
            variant="ghost"
            className="flex-col h-16 gap-1"
          >
            {isLoadingParts ? (
              <Spinner size="sm" />
            ) : (
              <Package className="h-5 w-5" />
            )}
            <span className="text-xs">Pièces</span>
          </Button>
        </div>
      </div>
    </div>
  )
}
