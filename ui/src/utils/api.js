const API_BASE_URL = 'http://localhost:8080/api';

// API utility functions
export const fetchBlocks = async (count = 10) => {
  const response = await fetch(`${API_BASE_URL}/blocks?count=${count}`);
  if (!response.ok) throw new Error('Failed to fetch blocks');
  return await response.json();
};

export const fetchBlock = async (hashOrHeight) => {
  const response = await fetch(`${API_BASE_URL}/blocks/${hashOrHeight}`);
  if (!response.ok) throw new Error('Failed to fetch block details');
  return await response.json();
};

export const fetchConsensusStatus = async () => {
  const response = await fetch(`${API_BASE_URL}/consensus`);
  if (!response.ok) throw new Error('Failed to fetch consensus status');
  return await response.json();
};

export const fetchGlobalStats = async () => {
  const response = await fetch(`${API_BASE_URL}/stats`);
  if (!response.ok) throw new Error('Failed to fetch global stats');
  return await response.json();
};

export const addTransaction = async (data) => {
  const response = await fetch(`${API_BASE_URL}/transactions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ data }),
  });
  
  if (!response.ok) throw new Error('Failed to add transaction');
  return await response.json();
};

// Helper functions
export const formatDate = (timestamp) => {
  const date = new Date(timestamp);
  return date.toLocaleString();
};

export const truncateHash = (hash, length = 8) => {
  if (!hash) return '';
  return `${hash.substring(0, length)}...${hash.substring(hash.length - length)}`;
};

export const getConsistencyLevelColor = (level) => {
  switch (level) {
    case 'Strong': return 'var(--success-color)';
    case 'Causal': return 'var(--secondary-color)';
    case 'Session': return 'var(--warning-color)';
    case 'Eventual': return 'var(--danger-color)';
    default: return 'var(--text-secondary)';
  }
};

export const getHealthStatusColor = (health) => {
  if (health > 0.8) return 'var(--success-color)';
  if (health > 0.5) return 'var(--warning-color)';
  return 'var(--danger-color)';
};