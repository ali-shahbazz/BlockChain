import React, { useState, useEffect } from 'react';
import { fetchGlobalStats, fetchConsensusStatus, getConsistencyLevelColor, getHealthStatusColor } from '../utils/api';

const Dashboard = () => {
  const [stats, setStats] = useState(null);
  const [consensus, setConsensus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const [statsData, consensusData] = await Promise.all([
          fetchGlobalStats(),
          fetchConsensusStatus()
        ]);
        setStats(statsData);
        setConsensus(consensusData);
        setLoading(false);
      } catch (err) {
        console.error('Error fetching dashboard data:', err);
        setError('Failed to load blockchain data. Please try again later.');
        setLoading(false);
      }
    };

    fetchData();
    // Poll for updates every 10 seconds
    const interval = setInterval(fetchData, 10000);
    
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="loading-container">
        <div className="spinner"></div>
        <p>Loading blockchain data...</p>
        <style jsx>{`
          .loading-container {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 300px;
          }
          
          .spinner {
            border: 4px solid rgba(0, 0, 0, 0.1);
            width: 36px;
            height: 36px;
            border-radius: 50%;
            border-left-color: var(--primary-color);
            animation: spin 1s linear infinite;
            margin-bottom: 1rem;
          }
          
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
        `}</style>
      </div>
    );
  }

  if (error) {
    return (
      <div className="error-container">
        <i className="fas fa-exclamation-triangle"></i>
        <p>{error}</p>
        <style jsx>{`
          .error-container {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 300px;
            color: var(--danger-color);
          }
          
          .error-container i {
            font-size: 2.5rem;
            margin-bottom: 1rem;
          }
        `}</style>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon">
            <i className="fas fa-cubes"></i>
          </div>
          <div className="stat-content">
            <h3>Blockchain Height</h3>
            <p className="stat-value">{stats?.height || 0}</p>
            <p className="stat-description">Total blocks in the chain</p>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">
            <i className="fas fa-exchange-alt"></i>
          </div>
          <div className="stat-content">
            <h3>Transactions</h3>
            <p className="stat-value">{stats?.totalTransactions || 0}</p>
            <p className="stat-description">Total transactions processed</p>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">
            <i className="fas fa-network-wired"></i>
          </div>
          <div className="stat-content">
            <h3>Consensus Status</h3>
            <p className="stat-value">{consensus?.currentProtocol || 'Hybrid'}</p>
            <p className="stat-description">Active consensus protocol</p>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">
            <i className="fas fa-server"></i>
          </div>
          <div className="stat-content">
            <h3>Validators</h3>
            <p className="stat-value">{consensus?.validatorCount || 0}</p>
            <p className="stat-description">Active validator nodes</p>
          </div>
        </div>
      </div>

      <div className="advanced-stats">
        <div className="advanced-stat-card">
          <h3>Adaptive Consistency</h3>
          <div className="consistency-level" style={{ color: getConsistencyLevelColor(consensus?.consistencyLevel) }}>
            <i className="fas fa-shield-alt"></i>
            <span>{consensus?.consistencyLevel || 'Unknown'}</span>
          </div>
          <div className="consistency-bar">
            <div className="consistency-levels">
              <span>Eventual</span>
              <span>Session</span>
              <span>Causal</span>
              <span>Strong</span>
            </div>
            <div className="bar-container">
              <div 
                className="bar-fill"
                style={{
                  width: `${consensus?.consistencyScore ? consensus.consistencyScore * 100 : 0}%`,
                  backgroundColor: getConsistencyLevelColor(consensus?.consistencyLevel)
                }}
              ></div>
            </div>
          </div>
        </div>

        <div className="advanced-stat-card">
          <h3>Network Health</h3>
          <div className="health-status" style={{ color: getHealthStatusColor(consensus?.networkHealth || 0) }}>
            <i className="fas fa-heartbeat"></i>
            <span>{consensus?.networkHealth ? `${(consensus.networkHealth * 100).toFixed(1)}%` : 'Unknown'}</span>
          </div>
          <div className="network-metrics">
            <div className="metric">
              <span>Latency</span>
              <span>{stats?.averageLatency ? `${stats.averageLatency.toFixed(2)}ms` : 'N/A'}</span>
            </div>
            <div className="metric">
              <span>Block Time</span>
              <span>{stats?.averageBlockTime ? `${stats.averageBlockTime.toFixed(2)}s` : 'N/A'}</span>
            </div>
            <div className="metric">
              <span>Fork Rate</span>
              <span>{stats?.forkRate !== undefined ? `${(stats.forkRate * 100).toFixed(2)}%` : 'N/A'}</span>
            </div>
          </div>
        </div>
      </div>

      <style jsx>{`
        .dashboard {
          padding: 2rem;
        }
        
        .stats-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
          gap: 1.5rem;
          margin-bottom: 2rem;
        }
        
        .stat-card {
          background-color: var(--card-background);
          border-radius: 8px;
          box-shadow: var(--shadow);
          padding: 1.5rem;
          display: flex;
          align-items: center;
          transition: transform 0.2s, box-shadow 0.2s;
        }
        
        .stat-card:hover {
          transform: translateY(-5px);
          box-shadow: 0 6px 12px rgba(0, 0, 0, 0.15);
        }
        
        .stat-icon {
          background-color: rgba(52, 152, 219, 0.1);
          color: var(--primary-color);
          width: 60px;
          height: 60px;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          margin-right: 1.5rem;
        }
        
        .stat-icon i {
          font-size: 1.8rem;
        }
        
        .stat-content h3 {
          font-size: 1rem;
          font-weight: 600;
          color: var(--text-secondary);
          margin: 0 0 0.5rem;
        }
        
        .stat-value {
          font-size: 1.8rem;
          font-weight: 700;
          margin: 0 0 0.5rem;
        }
        
        .stat-description {
          font-size: 0.85rem;
          color: var(--text-secondary);
          margin: 0;
        }
        
        .advanced-stats {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
          gap: 1.5rem;
          margin-bottom: 2rem;
        }
        
        .advanced-stat-card {
          background-color: var(--card-background);
          border-radius: 8px;
          box-shadow: var(--shadow);
          padding: 1.5rem;
        }
        
        .advanced-stat-card h3 {
          font-size: 1.2rem;
          font-weight: 600;
          margin: 0 0 1rem;
        }
        
        .consistency-level, .health-status {
          display: flex;
          align-items: center;
          font-size: 1.5rem;
          font-weight: 600;
          margin-bottom: 1.5rem;
        }
        
        .consistency-level i, .health-status i {
          margin-right: 0.75rem;
          font-size: 1.8rem;
        }
        
        .consistency-bar {
          margin-top: 1rem;
        }
        
        .consistency-levels {
          display: flex;
          justify-content: space-between;
          margin-bottom: 0.5rem;
          font-size: 0.75rem;
          color: var(--text-secondary);
        }
        
        .bar-container {
          height: 8px;
          background-color: var(--border-color);
          border-radius: 4px;
          overflow: hidden;
        }
        
        .bar-fill {
          height: 100%;
          border-radius: 4px;
          transition: width 0.5s ease;
        }
        
        .network-metrics {
          margin-top: 1.5rem;
        }
        
        .metric {
          display: flex;
          justify-content: space-between;
          padding: 0.75rem 0;
          border-bottom: 1px solid var(--border-color);
        }
        
        .metric:last-child {
          border-bottom: none;
        }
        
        @media (max-width: 768px) {
          .dashboard {
            padding: 1rem;
          }
          
          .advanced-stats {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
};

export default Dashboard;