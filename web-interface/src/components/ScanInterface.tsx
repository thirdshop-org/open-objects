import { useState, useEffect, useRef } from "react"
import { Button } from "./ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Camera, CameraOff, AlertCircle, CheckCircle, QrCode, ChevronLeft, Info } from "lucide-react"

// Import dynamique de QrScanner pour éviter les erreurs de build
let QrScanner: any = null

export default function ScanInterface() {
  const [isScanning, setIsScanning] = useState(false)
  const [status, setStatus] = useState("Initialisation...")
  const [error, setError] = useState<string | null>(null)
  const [isSupported, setIsSupported] = useState(true)
  const [showInfo, setShowInfo] = useState(false)
  const videoRef = useRef<HTMLVideoElement>(null)
  const scannerRef = useRef<any>(null)

  // Charger QrScanner dynamiquement
  useEffect(() => {
    const loadQrScanner = async () => {
      try {
        const script = document.createElement('script')
        script.src = 'http://127.0.0.1:8080/static/qr-scanner.min.js'
        script.onload = () => {
          QrScanner = (window as any).QrScanner
          setStatus("Appuyez pour scanner")
        }
        script.onerror = () => {
          setError("Impossible de charger la bibliothèque de scan QR")
          setIsSupported(false)
        }
        document.head.appendChild(script)
      } catch (err) {
        setError("Erreur lors du chargement du scanner QR")
        setIsSupported(false)
      }
    }

    loadQrScanner()
  }, [])

  // Fonction pour extraire l'ID ou le chemin depuis le texte du QR
  const extractIdOrPath = (text: string) => {
    // Format PRT-{id}
    let match = text.match(/^PRT-(\d+)$/i)
    if (match) return { id: match[1] }

    // Format LOC-{id}
    match = text.match(/^LOC-(\d+)$/i)
    if (match) return { locId: match[1] }

    // URL recycle://view/{id}
    match = text.match(/recycle:\/\/view\/(\d+)/i)
    if (match) return { id: match[1] }

    // URL HTTP avec /view/{id}
    match = text.match(/https?:\/\/[^\s]+\/view\/(\d+)/i)
    if (match) return { id: match[1] }

    // URL HTTP avec /location?id={id}
    match = text.match(/https?:\/\/[^\s]+\/location\?id=(\d+)/i)
    if (match) return { locId: match[1] }

    // Nombre simple comme ID
    match = text.match(/^\d+$/)
    if (match) return { id: match[0] }

    // Localisation webapp://loc?p={path}
    match = text.match(/webapp:\/\/loc\?p=([\w\-\._~\/]+)/i)
    if (match) return { path: decodeURIComponent(match[1]) }

    // URL avec /location?path={path}
    match = text.match(/\/location\?path=([\w\-\._~%\/]+)/i)
    if (match) return { path: decodeURIComponent(match[1]) }

    // Sinon, traiter comme path brut
    return { path: text }
  }

  // Démarrer le scan
  const startScanning = async () => {
    if (!QrScanner || !videoRef.current) {
      setError("Scanner QR non disponible")
      return
    }

    setError(null)
    setStatus("Démarrage de la caméra...")

    try {
      scannerRef.current = new QrScanner(
        videoRef.current,
        (result: string) => {
          setStatus(`QR détecté: ${result}`)
          const parsed = extractIdOrPath(result)

          if (parsed.id) {
            scannerRef.current?.stop()
            setIsScanning(false)
            setStatus(`Redirection vers la pièce...`)
            window.location.href = `/view/${parsed.id}`
          } else if (parsed.locId) {
            scannerRef.current?.stop()
            setIsScanning(false)
            setStatus(`Redirection vers la localisation...`)
            window.location.href = `/location?id=${parsed.locId}`
          } else if (parsed.path) {
            scannerRef.current?.stop()
            setIsScanning(false)
            setStatus(`Redirection vers le chemin...`)
            const encodedPath = encodeURIComponent(parsed.path)
            window.location.href = `/location?path=${encodedPath}`
          } else {
            setStatus(`QR non reconnu: ${result}`)
          }
        },
        {
          highlightScanRegion: true,
          highlightCodeOutline: true,
          returnDetailedScanResult: false,
        }
      )

      await scannerRef.current.start()
      setIsScanning(true)
      setStatus("Placez le QR code dans le cadre")
    } catch (err) {
      console.error("Erreur lors du démarrage du scanner:", err)
      setError(`Impossible de démarrer la caméra: ${err instanceof Error ? err.message : 'Erreur inconnue'}`)
      setIsSupported(false)
    }
  }

  // Arrêter le scan
  const stopScanning = () => {
    if (scannerRef.current) {
      scannerRef.current.stop()
      setIsScanning(false)
      setStatus("Scanner arrêté")
    }
  }

  // Vérifier le support de la caméra
  useEffect(() => {
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setIsSupported(false)
      setError("La capture vidéo n'est pas supportée par ce navigateur")
      setStatus("Caméra non disponible")
    }
  }, [])

  // Nettoyage à la fermeture
  useEffect(() => {
    return () => {
      if (scannerRef.current) {
        scannerRef.current.stop()
      }
    }
  }, [])

  return (
    <div className="min-h-screen bg-background flex flex-col">
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
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <QrCode className="h-6 w-6 text-primary shrink-0" />
            <h1 className="text-xl font-bold truncate">Scanner QR</h1>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setShowInfo(!showInfo)}
            className="shrink-0"
          >
            <Info className="h-5 w-5" />
          </Button>
        </div>
      </div>

      {/* Zone de scan plein écran */}
      <div className="flex-1 flex flex-col">
        {/* Vidéo plein écran */}
        <div className="flex-1 relative bg-black">
          {isSupported ? (
            <video
              ref={videoRef}
              playsInline
              className="w-full h-full object-cover"
            />
          ) : (
            <div className="w-full h-full flex flex-col items-center justify-center text-white p-6">
              <CameraOff className="h-16 w-16 mb-4 opacity-50" />
              <p className="text-center text-lg mb-2">Caméra non disponible</p>
              <p className="text-center text-sm opacity-75">
                La capture vidéo n'est pas supportée par ce navigateur
              </p>
            </div>
          )}

          {/* Overlay d'instructions */}
          {isScanning && (
            <div className="absolute inset-0 flex flex-col">
              {/* Zone de scan avec bordures */}
              <div className="flex-1 flex items-center justify-center p-6">
                <div className="relative w-full max-w-sm aspect-square">
                  {/* Coins du cadre */}
                  <div className="absolute top-0 left-0 w-12 h-12 border-t-4 border-l-4 border-primary rounded-tl-lg" />
                  <div className="absolute top-0 right-0 w-12 h-12 border-t-4 border-r-4 border-primary rounded-tr-lg" />
                  <div className="absolute bottom-0 left-0 w-12 h-12 border-b-4 border-l-4 border-primary rounded-bl-lg" />
                  <div className="absolute bottom-0 right-0 w-12 h-12 border-b-4 border-r-4 border-primary rounded-br-lg" />
                  
                  {/* Instructions au centre */}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="bg-black/60 text-white px-4 py-2 rounded-lg text-sm text-center backdrop-blur-sm">
                      Placez le QR code dans le cadre
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Barre de status */}
        <div className="bg-background border-t">
          <div className="p-4">
            <div className="flex items-center gap-3">
              {error ? (
                <AlertCircle className="h-5 w-5 text-destructive shrink-0" />
              ) : isScanning ? (
                <CheckCircle className="h-5 w-5 text-green-500 shrink-0" />
              ) : (
                <QrCode className="h-5 w-5 text-muted-foreground shrink-0" />
              )}
              <div className="min-w-0 flex-1">
                <p className={`text-sm font-medium truncate ${error ? 'text-destructive' : 'text-foreground'}`}>
                  {status}
                </p>
                {error && (
                  <p className="text-xs text-muted-foreground truncate mt-0.5">
                    {error}
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Bouton d'action fixe en bas */}
        <div className="bg-background border-t p-4 safe-area-inset-bottom">
          {!isScanning ? (
            <Button
              onClick={startScanning}
              disabled={!isSupported}
              size="lg"
              className="w-full h-14 text-base gap-2"
            >
              <Camera className="h-5 w-5" />
              Démarrer le scan
            </Button>
          ) : (
            <Button
              onClick={stopScanning}
              variant="destructive"
              size="lg"
              className="w-full h-14 text-base gap-2"
            >
              <CameraOff className="h-5 w-5" />
              Arrêter le scan
            </Button>
          )}
        </div>
      </div>

      {/* Panel d'informations (modal) */}
      {showInfo && (
        <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm">
          <div className="fixed inset-x-0 bottom-0 bg-background border-t rounded-t-2xl max-h-[80vh] overflow-y-auto animate-in slide-in-from-bottom-full duration-300">
            <div className="sticky top-0 bg-background border-b px-4 py-3 flex items-center justify-between">
              <h2 className="text-lg font-semibold">Informations</h2>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowInfo(false)}
              >
                <ChevronLeft className="h-5 w-5" />
              </Button>
            </div>

            <div className="p-4 space-y-6">
              {/* Formats supportés */}
              <div>
                <h3 className="font-semibold mb-3">Formats de QR supportés</h3>
                <div className="space-y-3">
                  <div className="flex items-start gap-3">
                    <Badge variant="outline" className="text-xs shrink-0 mt-0.5">PRT-123</Badge>
                    <span className="text-sm text-muted-foreground">Code pièce direct</span>
                  </div>
                  <div className="flex items-start gap-3">
                    <Badge variant="outline" className="text-xs shrink-0 mt-0.5">LOC-456</Badge>
                    <span className="text-sm text-muted-foreground">Code localisation</span>
                  </div>
                  <div className="flex items-start gap-3">
                    <Badge variant="outline" className="text-xs shrink-0 mt-0.5">123</Badge>
                    <span className="text-sm text-muted-foreground">ID numérique simple</span>
                  </div>
                  <div className="flex items-start gap-3">
                    <Badge variant="outline" className="text-xs shrink-0 mt-0.5">URL</Badge>
                    <span className="text-sm text-muted-foreground">Liens complets avec recycle:// ou http://</span>
                  </div>
                </div>
              </div>

              {/* Instructions */}
              <div>
                <h3 className="font-semibold mb-3">Comment utiliser ?</h3>
                <ol className="space-y-2 text-sm text-muted-foreground">
                  <li className="flex gap-2">
                    <span className="font-semibold text-foreground shrink-0">1.</span>
                    <span>Autorisez l'accès à la caméra si demandé</span>
                  </li>
                  <li className="flex gap-2">
                    <span className="font-semibold text-foreground shrink-0">2.</span>
                    <span>Placez le QR code dans le cadre lumineux</span>
                  </li>
                  <li className="flex gap-2">
                    <span className="font-semibold text-foreground shrink-0">3.</span>
                    <span>La détection est automatique</span>
                  </li>
                  <li className="flex gap-2">
                    <span className="font-semibold text-foreground shrink-0">4.</span>
                    <span>Vous serez redirigé automatiquement</span>
                  </li>
                </ol>
              </div>

              {/* Conseils */}
              <div className="bg-muted p-4 rounded-lg">
                <h3 className="font-semibold mb-2 text-sm">💡 Conseils</h3>
                <ul className="space-y-1 text-sm text-muted-foreground">
                  <li>• Assurez un bon éclairage</li>
                  <li>• Tenez le téléphone stable</li>
                  <li>• Gardez le QR code à plat</li>
                  <li>• Distance recommandée: 10-20cm</li>
                </ul>
              </div>
            </div>

            <div className="p-4 safe-area-inset-bottom">
              <Button
                onClick={() => setShowInfo(false)}
                variant="outline"
                size="lg"
                className="w-full"
              >
                Fermer
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
