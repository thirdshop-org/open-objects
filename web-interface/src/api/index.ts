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

export interface LocationSearchParams {
  search?: string;
  limit?: number;
  path?: string;
  id?: string;
}

export interface AddPartRequest {
  type?: string;
  name: string;
  loc?: string;
  props?: any;
  // Les photos sont envoyées comme des fichiers séparés
}

export interface AddPartResponse {
  id?: number;
  error?: string;
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
  getLocations: async (params?: LocationSearchParams): Promise<APIResult<LocationAPIResponse[]>> => {
    try {
      let url = `${API_BASE_URL}/api/locations`;

      if (params) {
        const searchParams = new URLSearchParams();
        if (params.search) searchParams.append('search', params.search);
        if (params.limit) searchParams.append('limit', params.limit.toString());
        if (params.path) searchParams.append('path', params.path);
        if (params.id) searchParams.append('id', params.id);

        const queryString = searchParams.toString();
        if (queryString) {
          url += `?${queryString}`;
        }
      }

      const response = await fetch(url);
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

  // Ajout d'une pièce - retourne [null, AddPartResponse] | [string, null]
  addPart: async (partData: AddPartRequest, photos?: File[]): Promise<APIResult<AddPartResponse>> => {
    try {
      const formData = new FormData();

      if (partData.type) formData.append('type', partData.type);
      formData.append('name', partData.name);
      if (partData.loc) formData.append('loc', partData.loc);
      if (partData.props) formData.append('props', JSON.stringify(partData.props));

      // Ajouter les photos
      if (photos) {
        photos.forEach((photo, index) => {
          formData.append(`photo_${index}`, photo);
        });
      }

      const response = await fetch(`${API_BASE_URL}/api/parts`, {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        const data = await response.json();
        return [null, data];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },

  // Recherche de pièces (partial HTML) - retourne [null, string] | [string, null]
  searchPartsPartial: async (query: string): Promise<APIResult<string>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/partials/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const html = await response.text();
        return [null, html];
      }
      return [`HTTP ${response.status}`, null];
    } catch (error) {
      return [error instanceof Error ? error.message : 'Unknown error', null];
    }
  },
};
