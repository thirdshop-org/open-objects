// API Base URL
const API_BASE_URL = 'http://127.0.0.1:8080';

// Types pour les réponses API
export interface PartAPIResponse {
  id: number;
  type: string;
  name: string;
  props: any;
  location?: string;
  source?: string;
}

export interface LocationAPIResponse {
  id: number;
  name: string;
  parent_id?: number;
  loc_type: string;
  description: string;
  path: string;
}

export interface SearchResult {
  parts: PartAPIResponse[];
  total: number;
}

// Définition des fonctions API avec tuples pour les types de retour
export const api = {
  // Health check - retourne [success: boolean, error?: string]
  health: async (): Promise<[boolean, string?]> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/health`);
      if (response.ok) {
        return [true];
      }
      return [false, `HTTP ${response.status}`];
    } catch (error) {
      return [false, error instanceof Error ? error.message : 'Unknown error'];
    }
  },

  // Recherche de pièces - retourne [résultats: SearchResult | null, error?: string]
  search: async (query: string): Promise<[SearchResult | null, string?]> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const data = await response.json();
        return [data];
      }
      return [null, `HTTP ${response.status}`];
    } catch (error) {
      return [null, error instanceof Error ? error.message : 'Unknown error'];
    }
  },

  // Récupération de toutes les pièces - retourne [pièces: PartAPIResponse[] | null, error?: string]
  getParts: async (): Promise<[PartAPIResponse[] | null, string?]> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/search`);
      if (response.ok) {
        const data = await response.json();
        return [data.parts || []];
      }
      return [null, `HTTP ${response.status}`];
    } catch (error) {
      return [null, error instanceof Error ? error.message : 'Unknown error'];
    }
  },

  // Récupération des localisations - retourne [localisations: LocationAPIResponse[] | null, error?: string]
  getLocations: async (): Promise<[LocationAPIResponse[] | null, string?]> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/locations`);
      if (response.ok) {
        const data = await response.json();
        return [data];
      }
      return [null, `HTTP ${response.status}`];
    } catch (error) {
      return [null, error instanceof Error ? error.message : 'Unknown error'];
    }
  },

  // Recherche fédérée - retourne [résultats: SearchResult | null, error?: string]
  federatedSearch: async (query: string): Promise<[SearchResult | null, string?]> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/federated/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const data = await response.json();
        return [data];
      }
      return [null, `HTTP ${response.status}`];
    } catch (error) {
      return [null, error instanceof Error ? error.message : 'Unknown error'];
    }
  },

  // Récupération des champs de template - retourne [champs: any | null, error?: string]
  getTemplateFields: async (): Promise<[any | null, string?]> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/template-fields`);
      if (response.ok) {
        const data = await response.json();
        return [data];
      }
      return [null, `HTTP ${response.status}`];
    } catch (error) {
      return [null, error instanceof Error ? error.message : 'Unknown error'];
    }
  },
};
