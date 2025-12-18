import { useState, useEffect, useRef } from "react"
import { Button } from "./ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { Badge } from "./ui/badge"
import { Camera, CameraOff, AlertCircle, CheckCircle, QrCode } from "lucide-react"

// Import dynamique de QrScanner pour éviter les erreurs de build
let QrScanner: any = null

export default function ScanInterface() {
  const [isScanning, setIsScanning] = useState(false)
  const [status, setStatus] = useState("Initialisation...")
  const [error, setError] = useState<string | null>(null)
  const [isSupported, setIsSupported] = useState(true)
  const videoRef = useRef<HTMLVideoElement>(null)
  const scannerRef = useRef<any>(null)

  // Charger QrScanner dynamiquement
  useEffect(() => {
    const loadQrScanner = async () => {
      try {
        // Charger le script qr-scanner depuis l'API backend
        const script = document.createElement('script')
        script.src = 'http://127.0.0.1:8080/static/qr-scanner.min.js'
        script.onload = () => {
          QrScanner = (window as any).QrScanner
          setStatus("Scanner prêt")
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
            setStatus(`Redirection vers la pièce ${parsed.id}...`)
            // Rediriger vers la page de la pièce
            window.location.href = `/view/${parsed.id}`
          } else if (parsed.locId) {
            scannerRef.current?.stop()
            setIsScanning(false)
            setStatus(`Redirection vers la localisation ${parsed.locId}...`)
            // Rediriger vers la page de localisation
            window.location.href = `/location?id=${parsed.locId}`
          } else if (parsed.path) {
            scannerRef.current?.stop()
            setIsScanning(false)
            setStatus(`Redirection vers le chemin ${parsed.path}...`)
            // Rediriger vers la page de localisation avec path
            const encodedPath = encodeURIComponent(parsed.path)
            window.location.href = `/location?path=${encodedPath}`
          } else {
            setStatus(`QR détecté mais non reconnu: ${result}`)
          }
        },
        {
          highlightScanRegion: true,
          highlightCodeOutline: true,
        }
      )

      await scannerRef.current.start()
      setIsScanning(true)
      setStatus("Caméra active, scannez un QR code...")
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

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted/20">
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <QrCode className="h-12 w-12 text-primary" />
            <h1 className="text-4xl font-bold">Scanner QR Code</h1>
          </div>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Scannez un QR code pour accéder rapidement à une pièce ou une localisation
          </p>
        </div>

        {/* Zone de scan */}
        <div className="grid gap-6 md:grid-cols-1 lg:grid-cols-3">
          {/* Vidéo et contrôles */}
          <div className="lg:col-span-2">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Camera className="h-5 w-5" />
                  Caméra
                </CardTitle>
                <CardDescription>
                  Placez le QR code dans le cadre pour le scanner automatiquement
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Vidéo */}
                <div className="relative">
                  <video
                    ref={videoRef}
                    playsInline
                    className="w-full max-w-md mx-auto border border-border rounded-lg shadow-sm"
                    style={{ display: isSupported ? 'block' : 'none' }}
                  />
                  {!isSupported && (
                    <div className="w-full max-w-md mx-auto h-64 bg-muted rounded-lg flex items-center justify-center">
                      <div className="text-center">
                        <CameraOff className="h-12 w-12 text-muted-foreground mx-auto mb-2" />
                        <p className="text-muted-foreground">Caméra non disponible</p>
                      </div>
                    </div>
                  )}
                </div>

                {/* Contrôles */}
                <div className="flex gap-2 justify-center">
                  {!isScanning ? (
                    <Button
                      onClick={startScanning}
                      disabled={!isSupported}
                      className="gap-2"
                    >
                      <Camera className="h-4 w-4" />
                      Démarrer le scan
                    </Button>
                  ) : (
                    <Button
                      onClick={stopScanning}
                      variant="outline"
                      className="gap-2"
                    >
                      <CameraOff className="h-4 w-4" />
                      Arrêter
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Status et informations */}
          <div className="space-y-6">
            {/* Status */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {error ? (
                    <AlertCircle className="h-5 w-5 text-destructive" />
                  ) : isScanning ? (
                    <CheckCircle className="h-5 w-5 text-green-500" />
                  ) : (
                    <QrCode className="h-5 w-5" />
                  )}
                  Status
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <p className={`text-sm ${error ? 'text-destructive' : 'text-foreground'}`}>
                    {status}
                  </p>
                  {error && (
                    <p className="text-xs text-muted-foreground">
                      {error}
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Formats supportés */}
            <Card>
              <CardHeader>
                <CardTitle>Formats de QR supportés</CardTitle>
                <CardDescription>
                  Le scanner reconnaît différents formats de QR codes
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">PRT-123</Badge>
                    <span className="text-muted-foreground">Pièce</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">LOC-456</Badge>
                    <span className="text-muted-foreground">Localisation</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">recycle://view/123</Badge>
                    <span className="text-muted-foreground">URL pièce</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">webapp://loc?p=path</Badge>
                    <span className="text-muted-foreground">URL localisation</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">123</Badge>
                    <span className="text-muted-foreground">ID numérique</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Instructions */}
            <Card>
              <CardHeader>
                <CardTitle>Instructions</CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="text-sm text-muted-foreground space-y-1">
                  <li>• Autorisez l'accès à la caméra</li>
                  <li>• Placez le QR code dans le cadre</li>
                  <li>• Le scanner détecte automatiquement</li>
                  <li>• Vous serez redirigé vers la page appropriée</li>
                </ul>
              </CardContent>
            </Card>
          </div>
        </div>

        {/* Message d'erreur global si non supporté */}
        {!isSupported && (
          <Card className="mt-6 border-destructive">
            <CardContent className="pt-6">
              <div className="flex items-center gap-3">
                <AlertCircle className="h-5 w-5 text-destructive flex-shrink-0" />
                <div>
                  <h3 className="font-semibold text-destructive">Caméra non disponible</h3>
                  <p className="text-sm text-muted-foreground mt-1">
                    La capture vidéo n'est pas supportée par ce navigateur.
                    Essayez Chrome/Firefox récent ou scannez avec l'appareil photo et ouvrez le lien généré.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
