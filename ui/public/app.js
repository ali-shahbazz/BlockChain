// Global state
const state = {
    blocks: [],
    blockchain: {
        height: 0,
        transactions: 0
    },
    consensus: {
        state: '',
        leadingVoteCount: 0,
        consistencyLevel: '',
        networkHealth: 0,
        partitionProbability: 0
    },
    network: {
        shardCount: 0,
        activeConflicts: 0,
        resolvedConflicts: 0,
        avgEntropy: 0
    },
    history: {
        partitionProbability: [],
        consistencyLevels: []
    },
    currentPage: 'dashboard',
    selectedBlock: null
};

// Charts
let networkChart = null;
let consistencyChart = null;
let partitionChart = null;

// DOM Elements
const dashboardView = document.getElementById('dashboard-view');
const blocksView = document.getElementById('blocks-view');
const blockDetailsView = document.getElementById('block-details-view');
const consensusView = document.getElementById('consensus-view');
const networkView = document.getElementById('network-view');

// URL for the API endpoints
const API_BASE_URL = 'http://localhost:8080/api';

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    setupNavigation();
    setupEventListeners();
    initializeDashboard();
    
    // Initialize Bootstrap tooltips
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });
    
    // Initialize Bootstrap modal
    const transactionModal = new bootstrap.Modal(document.getElementById('addTransactionModal'));
    
    // Start periodic data updates
    setInterval(updateData, 5000);
});

function setupNavigation() {
    // Dashboard navigation
    document.getElementById('dashboardNav').addEventListener('click', () => {
        showView('dashboard');
    });
    
    // Blocks navigation
    document.getElementById('blocksNav').addEventListener('click', () => {
        showView('blocks');
        loadBlocks();
    });
    
    // Consensus navigation
    document.getElementById('consensusNav').addEventListener('click', () => {
        showView('consensus');
        loadConsensusStatus();
    });
    
    // Network navigation
    document.getElementById('networkNav').addEventListener('click', () => {
        showView('network');
        loadNetworkStatus();
    });
    
    // Back to blocks button
    document.getElementById('back-to-blocks').addEventListener('click', () => {
        showView('blocks');
    });
}

function setupEventListeners() {
    // Add transaction button
    document.getElementById('addTransactionBtn').addEventListener('click', () => {
        const modal = new bootstrap.Modal(document.getElementById('addTransactionModal'));
        modal.show();
    });
    
    // Submit transaction button
    document.getElementById('submitTransaction').addEventListener('click', () => {
        const data = document.getElementById('transactionData').value;
        if (data) {
            submitTransaction(data);
        }
    });
    
    // Search button
    document.getElementById('search-btn').addEventListener('click', () => {
        const searchTerm = document.getElementById('block-search').value;
        if (searchTerm) {
            searchBlocks(searchTerm);
        }
    });
}

function showView(viewName) {
    // Hide all views
    dashboardView.classList.add('d-none');
    blocksView.classList.add('d-none');
    blockDetailsView.classList.add('d-none');
    consensusView.classList.add('d-none');
    networkView.classList.add('d-none');
    
    // Show the requested view
    switch (viewName) {
        case 'dashboard':
            dashboardView.classList.remove('d-none');
            updateDashboard();
            break;
        case 'blocks':
            blocksView.classList.remove('d-none');
            break;
        case 'block-details':
            blockDetailsView.classList.remove('d-none');
            break;
        case 'consensus':
            consensusView.classList.remove('d-none');
            break;
        case 'network':
            networkView.classList.remove('d-none');
            break;
    }
    
    // Update navigation active state
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
    });
    
    const activeNav = document.getElementById(`${viewName}Nav`);
    if (activeNav) {
        activeNav.classList.add('active');
    }
    
    state.currentPage = viewName;
}

async function initializeDashboard() {
    try {
        // Load initial data
        await Promise.all([
            fetchBlockchainStatus(),
            fetchConsensusStatus(),
            fetchNetworkStatus(),
            loadBlocks()
        ]);
        
        // Initialize charts
        initializeNetworkChart();
        initializeConsistencyChart();
        initializePartitionChart();
        
        // Update the dashboard
        updateDashboard();
    } catch (error) {
        console.error('Error initializing dashboard:', error);
    }
}

async function updateData() {
    try {
        // Fetch updated data based on current page
        await fetchBlockchainStatus();
        
        if (state.currentPage === 'dashboard') {
            updateDashboard();
        } else if (state.currentPage === 'blocks') {
            await loadBlocks();
        } else if (state.currentPage === 'consensus') {
            await loadConsensusStatus();
        } else if (state.currentPage === 'network') {
            await loadNetworkStatus();
        } else if (state.currentPage === 'block-details' && state.selectedBlock) {
            await loadBlockDetails(state.selectedBlock);
        }
    } catch (error) {
        console.error('Error updating data:', error);
    }
}

function updateDashboard() {
    // Update blockchain stats
    document.getElementById('blockchain-height').textContent = state.blockchain.height;
    document.getElementById('transactions-count').textContent = state.blockchain.transactions;
    
    // Update consistency level
    document.getElementById('consistency-level').textContent = getConsistencyLevelText(state.consensus.consistencyLevel);
    
    // Update network health
    document.getElementById('network-health').textContent = `${Math.round(state.consensus.networkHealth * 100)}%`;
    
    // Update recent blocks table
    const recentBlocksTableBody = document.querySelector('#recent-blocks-table tbody');
    recentBlocksTableBody.innerHTML = '';
    
    // Display only 5 most recent blocks
    const recentBlocks = state.blocks.slice(0, 5);
    
    recentBlocks.forEach(block => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${block.height}</td>
            <td class="hash-display">${shortenHash(block.hash)}</td>
            <td>${block.transactionCount}</td>
            <td>${formatTimestamp(block.timestamp)}</td>
        `;
        row.addEventListener('click', () => {
            state.selectedBlock = block.hash;
            loadBlockDetails(block.hash);
            showView('block-details');
        });
        recentBlocksTableBody.appendChild(row);
    });
    
    // Update network chart
    updateNetworkChart();
}

async function loadBlocks() {
    try {
        const response = await fetch(`${API_BASE_URL}/blocks`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        state.blocks = data.blocks || [];
        
        // Update blocks table
        updateBlocksTable();
    } catch (error) {
        console.error('Error loading blocks:', error);
    }
}

function updateBlocksTable() {
    const blocksTableBody = document.querySelector('#blocks-table tbody');
    blocksTableBody.innerHTML = '';
    
    state.blocks.forEach(block => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${block.height}</td>
            <td class="hash-display">${shortenHash(block.hash)}</td>
            <td class="hash-display">${shortenHash(block.prevHash)}</td>
            <td>${block.transactionCount}</td>
            <td>${formatTimestamp(block.timestamp)}</td>
            <td>
                <span class="badge ${getEntropyBadgeClass(block.entropy)}">${block.entropy ? block.entropy.toFixed(2) : 'N/A'}</span>
            </td>
            <td>
                <button class="btn btn-sm btn-primary view-block-btn" data-hash="${block.hash}">View</button>
            </td>
        `;
        blocksTableBody.appendChild(row);
    });
    
    // Add event listeners to view buttons with a more specific selector
    document.querySelectorAll('.view-block-btn').forEach(button => {
        button.addEventListener('click', function(e) {
            // Using a regular function to ensure 'this' refers to the button
            const hash = this.getAttribute('data-hash');
            console.log('View button clicked for hash:', hash);
            if (hash) {
                state.selectedBlock = hash;
                loadBlockDetails(hash);
                showView('block-details');
            } else {
                console.error('No hash attribute found on view button');
            }
        });
    });
}

async function loadBlockDetails(hash) {
    try {
        // Show loading indicator
        document.getElementById('block-height').textContent = 'Loading...';
        document.getElementById('block-hash').textContent = 'Loading...';
        
        console.log(`Fetching block details for hash: ${hash}`);
        const response = await fetch(`${API_BASE_URL}/blocks/${hash}`);
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const block = await response.json();
        console.log('Block details received:', block);
        
        // Update block details
        document.getElementById('block-height').textContent = block.height;
        document.getElementById('block-hash').textContent = block.hash;
        document.getElementById('block-prev-hash').textContent = block.prevHash || 'Genesis Block';
        document.getElementById('block-merkle-root').textContent = block.merkleRoot || 'N/A';
        document.getElementById('block-timestamp').textContent = formatTimestamp(block.timestamp);
        document.getElementById('block-nonce').textContent = block.nonce || '0';
        document.getElementById('block-entropy').textContent = block.entropy ? block.entropy.toFixed(4) : 'N/A';
        document.getElementById('block-shard').textContent = block.shardId || 'Main';
        
        // Update transactions table
        const txTableBody = document.querySelector('#block-transactions-table tbody');
        txTableBody.innerHTML = '';
        
        if (block.transactions && block.transactions.length > 0) {
            block.transactions.forEach(tx => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td class="hash-display">${shortenHash(tx.id)}</td>
                    <td>${formatTimestamp(tx.timestamp)}</td>
                    <td>${tx.data ? atob(tx.data) : 'N/A'}</td>
                    <td>${tx.type || 'Standard'}</td>
                `;
                txTableBody.appendChild(row);
            });
        } else {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="4" class="text-center">No transactions in this block</td>';
            txTableBody.appendChild(row);
        }
    } catch (error) {
        console.error('Error loading block details:', error);
        // Display error message in the UI
        document.getElementById('block-height').textContent = 'Error';
        document.getElementById('block-hash').textContent = 'Failed to load block details: ' + error.message;
        
        // Clear other fields
        const fields = ['block-prev-hash', 'block-merkle-root', 'block-timestamp', 'block-nonce', 'block-entropy', 'block-shard'];
        fields.forEach(id => {
            document.getElementById(id).textContent = '-';
        });
        
        // Clear transactions table
        const txTableBody = document.querySelector('#block-transactions-table tbody');
        txTableBody.innerHTML = '';
        const row = document.createElement('tr');
        row.innerHTML = '<td colspan="4" class="text-center text-danger">Failed to load transactions</td>';
        txTableBody.appendChild(row);
    }
}

async function loadConsensusStatus() {
    try {
        const response = await fetch(`${API_BASE_URL}/consensus`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        // Update consensus state
        state.consensus = {
            state: data.consensusState || 'Unknown',
            leadingVoteCount: data.leadingVoteCount || 0,
            consistencyLevel: data.consistencyLevel || 'strong',
            networkHealth: data.networkHealth || 0,
            partitionProbability: data.partitionProbability || 0
        };
        
        // Record history for charts
        state.history.consistencyLevels.push({
            timestamp: new Date(),
            level: state.consensus.consistencyLevel
        });
        
        // Keep history limited to 20 entries
        if (state.history.consistencyLevels.length > 20) {
            state.history.consistencyLevels.shift();
        }
        
        // Update UI
        document.getElementById('consensus-state').textContent = state.consensus.state;
        document.getElementById('leading-vote-count').textContent = state.consensus.leadingVoteCount;
        document.getElementById('consistency-level-detailed').textContent = getConsistencyLevelText(state.consensus.consistencyLevel);
        document.getElementById('network-health-detailed').textContent = `${Math.round(state.consensus.networkHealth * 100)}%`;
        document.getElementById('partition-probability').textContent = `${Math.round(state.consensus.partitionProbability * 100)}%`;
        
        // Update consistency chart
        updateConsistencyChart();
    } catch (error) {
        console.error('Error loading consensus status:', error);
    }
}

async function loadNetworkStatus() {
    try {
        const response = await fetch(`${API_BASE_URL}/network`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        // Update network state
        state.network = {
            shardCount: data.shardCount || 1,
            activeConflicts: data.activeConflicts || 0,
            resolvedConflicts: data.resolvedConflicts || 0,
            avgEntropy: data.averageEntropy || 0
        };
        
        // Record partition probability history
        state.history.partitionProbability.push({
            timestamp: new Date(),
            probability: state.consensus.partitionProbability
        });
        
        // Keep history limited to 20 entries
        if (state.history.partitionProbability.length > 20) {
            state.history.partitionProbability.shift();
        }
        
        // Update UI
        document.getElementById('shard-count').textContent = state.network.shardCount;
        document.getElementById('active-conflicts').textContent = state.network.activeConflicts;
        document.getElementById('resolved-conflicts').textContent = state.network.resolvedConflicts;
        document.getElementById('avg-entropy').textContent = state.network.avgEntropy.toFixed(4);
        
        // Update partition chart
        updatePartitionChart();
    } catch (error) {
        console.error('Error loading network status:', error);
    }
}

async function fetchBlockchainStatus() {
    try {
        const response = await fetch(`${API_BASE_URL}/status`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        state.blockchain = {
            height: data.height || 0,
            transactions: data.transactionCount || 0
        };
    } catch (error) {
        console.error('Error fetching blockchain status:', error);
    }
}

async function fetchConsensusStatus() {
    try {
        const response = await fetch(`${API_BASE_URL}/consensus`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        state.consensus = {
            state: data.consensusState || 'Unknown',
            leadingVoteCount: data.leadingVoteCount || 0,
            consistencyLevel: data.consistencyLevel || 'strong',
            networkHealth: data.networkHealth || 0,
            partitionProbability: data.partitionProbability || 0
        };
    } catch (error) {
        console.error('Error fetching consensus status:', error);
    }
}

async function fetchNetworkStatus() {
    try {
        const response = await fetch(`${API_BASE_URL}/network`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        state.network = {
            shardCount: data.shardCount || 1,
            activeConflicts: data.activeConflicts || 0,
            resolvedConflicts: data.resolvedConflicts || 0,
            avgEntropy: data.averageEntropy || 0
        };
    } catch (error) {
        console.error('Error fetching network status:', error);
    }
}

async function submitTransaction(data) {
    try {
        const response = await fetch(`${API_BASE_URL}/transactions`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ data })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        // Close modal and clear form
        const modal = bootstrap.Modal.getInstance(document.getElementById('addTransactionModal'));
        modal.hide();
        document.getElementById('transactionData').value = '';
        
        // Show success message
        alert('Transaction submitted successfully!');
        
        // Refresh data
        await updateData();
    } catch (error) {
        console.error('Error submitting transaction:', error);
        alert(`Error submitting transaction: ${error.message}`);
    }
}

async function searchBlocks(searchTerm) {
    try {
        // Try to search by hash first
        if (searchTerm.length > 8) {
            // Might be a hash
            const blocksByHash = state.blocks.filter(block => 
                block.hash && block.hash.toLowerCase().includes(searchTerm.toLowerCase())
            );
            
            if (blocksByHash.length > 0) {
                // Found blocks by hash
                state.selectedBlock = blocksByHash[0].hash;
                loadBlockDetails(blocksByHash[0].hash);
                showView('block-details');
                return;
            }
        }
        
        // Try to search by height
        const height = parseInt(searchTerm);
        if (!isNaN(height)) {
            const blockByHeight = state.blocks.find(block => block.height === height);
            if (blockByHeight) {
                state.selectedBlock = blockByHeight.hash;
                loadBlockDetails(blockByHeight.hash);
                showView('block-details');
                return;
            }
        }
        
        // If no match found
        alert('No matching block found. Please try a different search term.');
    } catch (error) {
        console.error('Error searching blocks:', error);
    }
}

// Chart initialization and updates
function initializeNetworkChart() {
    const ctx = document.getElementById('network-chart').getContext('2d');
    networkChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Network Health',
                data: [],
                borderColor: 'rgba(75, 192, 192, 1)',
                tension: 0.1,
                fill: false
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100,
                    title: {
                        display: true,
                        text: 'Health %'
                    }
                }
            }
        }
    });
}

function updateNetworkChart() {
    if (!networkChart) return;
    
    // Generate labels for the last 10 intervals (assuming 5s intervals)
    const labels = Array.from({ length: 10 }, (_, i) => {
        const now = new Date();
        return `${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds() - (i * 5)).padStart(2, '0')}`;
    }).reverse();
    
    // Generate network health data (if we don't have historical data, use current data repeated)
    const healthData = Array.from({ length: 10 }, () => Math.round(state.consensus.networkHealth * 100));
    
    networkChart.data.labels = labels;
    networkChart.data.datasets[0].data = healthData;
    networkChart.update();
}

function initializeConsistencyChart() {
    const ctx = document.getElementById('consistency-chart').getContext('2d');
    consistencyChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: ['Strong', 'Causal', 'Session', 'Eventual'],
            datasets: [{
                label: 'Time in Each Consistency Level',
                data: [0, 0, 0, 0],
                backgroundColor: [
                    'rgba(54, 162, 235, 0.6)',
                    'rgba(75, 192, 192, 0.6)',
                    'rgba(255, 206, 86, 0.6)',
                    'rgba(255, 99, 132, 0.6)'
                ],
                borderColor: [
                    'rgba(54, 162, 235, 1)',
                    'rgba(75, 192, 192, 1)',
                    'rgba(255, 206, 86, 1)',
                    'rgba(255, 99, 132, 1)'
                ],
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Time (s)'
                    }
                }
            }
        }
    });
}

function updateConsistencyChart() {
    if (!consistencyChart || !state.history.consistencyLevels.length) return;
    
    // Count occurrences of each consistency level
    const counts = {
        strong: 0,
        causal: 0,
        session: 0,
        eventual: 0
    };
    
    state.history.consistencyLevels.forEach(item => {
        counts[item.level]++;
    });
    
    // Update chart data
    consistencyChart.data.datasets[0].data = [
        counts.strong,
        counts.causal,
        counts.session,
        counts.eventual
    ];
    
    consistencyChart.update();
}

function initializePartitionChart() {
    const ctx = document.getElementById('partition-chart').getContext('2d');
    partitionChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Partition Probability',
                data: [],
                borderColor: 'rgba(255, 99, 132, 1)',
                backgroundColor: 'rgba(255, 99, 132, 0.2)',
                tension: 0.1,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100,
                    title: {
                        display: true,
                        text: 'Probability %'
                    }
                }
            }
        }
    });
}

function updatePartitionChart() {
    if (!partitionChart || !state.history.partitionProbability.length) return;
    
    // Generate labels and data
    const labels = state.history.partitionProbability.map(item => {
        const time = item.timestamp;
        return `${time.getHours()}:${String(time.getMinutes()).padStart(2, '0')}:${String(time.getSeconds()).padStart(2, '0')}`;
    });
    
    const data = state.history.partitionProbability.map(item => Math.round(item.probability * 100));
    
    partitionChart.data.labels = labels;
    partitionChart.data.datasets[0].data = data;
    partitionChart.update();
}

// Helper functions
function shortenHash(hash) {
    if (!hash) return 'N/A';
    if (hash.length <= 16) return hash;
    return `${hash.substring(0, 8)}...${hash.substring(hash.length - 8)}`;
}

function formatTimestamp(timestamp) {
    if (!timestamp) return 'N/A';
    
    // Handle string timestamps that might need parsing
    if (typeof timestamp === 'string') {
        // If timestamp is already in ISO format
        if (timestamp.includes('T') || timestamp.includes('-')) {
            return new Date(timestamp).toLocaleString();
        }
        // If timestamp is a string number
        timestamp = parseInt(timestamp);
    }
    
    // Check if timestamp is in milliseconds or seconds
    // If timestamp is too small to be in milliseconds (before year 2000), multiply by 1000
    if (timestamp < 1600000000000) {
        timestamp = timestamp * 1000;
    }
    
    const date = new Date(timestamp);
    
    // Check if date is valid
    if (isNaN(date.getTime())) {
        return 'Invalid Date';
    }
    
    return date.toLocaleString();
}

function getConsistencyLevelText(level) {
    switch (level) {
        case 'strong':
            return 'Strong';
        case 'causal':
            return 'Causal';
        case 'session':
            return 'Session';
        case 'eventual':
            return 'Eventual';
        default:
            return level ? level.charAt(0).toUpperCase() + level.slice(1) : 'Unknown';
    }
}

function getEntropyBadgeClass(entropy) {
    if (!entropy) return 'bg-secondary';
    if (entropy < 0.3) return 'bg-danger';
    if (entropy < 0.5) return 'bg-warning';
    if (entropy < 0.7) return 'bg-info';
    return 'bg-success';
}