import axios from 'axios';
import { ResourceNode, ResourceRelationship } from '../types';

const API_BASE_URL = '/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

export const apiService = {
  // Health check
  async healthCheck(): Promise<{ status: string; message: string }> {
    const response = await api.get('/health');
    return response.data;
  },

  // Get all namespaces
  async getNamespaces(): Promise<string[]> {
    const response = await api.get('/namespaces');
    return response.data;
  },

  // Get resources by type
  async getResourcesByType(resourceType: string, namespace?: string): Promise<ResourceNode[]> {
    const params = namespace ? { namespace } : {};
    const response = await api.get(`/resources/${resourceType}`, { params });
    return response.data;
  },

  // Get resource children (ownerReference relationships)
  async getResourceChildren(resourceType: string, resourceName: string, namespace?: string): Promise<ResourceRelationship> {
    const params = namespace ? { namespace } : {};
    const response = await api.get(`/resources/${resourceType}/${resourceName}/children`, { params });
    return response.data;
  },
};

export default apiService;
