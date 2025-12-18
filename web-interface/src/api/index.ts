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


// Types pour les réponses API avec union discriminée
// Pattern: [error, null] | [null, T]
// Exemple d'utilisation:
// const [error, result] = await api.search("query")
// if (error) { /* gérer l'erreur */ } else { /* utiliser result */ }
type APIResult<T> = [error: string, result: null] | [error: null, result: T]

// Définition des fonctions API avec tuples discriminés
export const api = {
  // Health check - retourne [null, boolean] | [string, null]
  health: async (): Promise<APIResult<boolean>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/health`);
      if (response.ok) {
        return [null, true];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },

  // Recherche de pièces - retourne [null, PartAPIResponse[]] | [string, null]
  search: async (query: string): Promise<APIResult<PartAPIResponse[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const data = await response.json();
        return [null, data];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },

  // Récupération de toutes les pièces - retourne [null, PartAPIResponse[]] | [string, null]
  getParts: async (): Promise<APIResult<PartAPIResponse[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/search`);
      if (response.ok) {
        const data = await response.json();
        return [null, data];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },

  // Récupération des localisations - retourne [null, LocationAPIResponse[]] | [string, null]
  getLocations: async (): Promise<APIResult<LocationAPIResponse[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/locations`);
      if (response.ok) {
        const data = await response.json();
        return [null, data];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },

  // Recherche fédérée - retourne [null, PartAPIResponse[]] | [string, null]
  federatedSearch: async (query: string): Promise<APIResult<PartAPIResponse[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/federated/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const data = await response.json();
        return [null, data];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },

  // Récupération des champs de template - retourne [null, any] | [string, null]
  getTemplateFields: async (): Promise<APIResult<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/template-fields`);
      if (response.ok) {
        const data = await response.json();
        return [null, data];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },
};
