import React, { useState, useEffect } from 'react';
import { fetchBlocks, fetchBlock, formatDate, truncateHash } from '../utils/api';

const Blocks = () => {
  const [blocks, setBlocks] = useState([]);
  const [selectedBlock, setSelectedBlock] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchBlockData = async () => {
      try {
        setLoading(true);
        const data = await fetchBlocks(10);
        setBlocks(data);
        setLoading(false);
      } catch (err) {
        console.error('Error fetching blocks:', err);
        setError('Failed to load blockchain blocks. Please try again later.');
        setLoading(false);
      }
    };

    fetchBlockData();
    // Poll for updates every 30 seconds
    const interval = setInterval(fetchBlockData, 30000);
    
    return () => clearInterval(interval);
  }, []);

  const handleBlockClick = async (height) => {
    try {
      const blockData = await fetchBlock(height);
      setSelectedBlock(blockData);
    } catch (err) {
      console.error('Error fetching block details:', err);
      alert('Failed to load block details. Please try again.');
    }
  };

  const closeBlockDetails = () => {
    setSelectedBlock(null);
  };

  if (loading && blocks.length === 0) {
    return (
      <div className="loading-container">
        <div className="spinner"></div>
        <p>Loading blockchain blocks...</p>
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
    <div className="blocks-container">
      <h2>Blockchain Blocks</h2>
      
      <div className="blocks-timeline">
        {blocks.map((block, index) => (
          <div 
            key={block.height} 
            className="block-item"
            onClick={() => handleBlockClick(block.height)}
          >
            <div className="block-icon">
              <i className="fas fa-cube"></i>
            </div>
            <div className="block-connector">
              {index < blocks.length - 1 && <div className="connector-line"></div>}
            </div>
            <div className="block-card">
              <div className="block-header">
                <span className="block-height">Block #{block.height}</span>
                <span className="block-time">{formatDate(block.timestamp)}</span>
              </div>
              <div className="block-body">
                <div className="block-info">
                  <div className="info-item">
                    <span className="label">Hash:</span>
                    <span className="value hash">{truncateHash(block.hash)}</span>
                  </div>
                  <div className="info-item">
                    <span className="label">Prev Hash:</span>
                    <span className="value hash">{truncateHash(block.previousHash)}</span>
                  </div>
                  <div className="info-item">
                    <span className="label">Transactions:</span>
                    <span className="value">{block.transactionCount}</span>
                  </div>
                  {block.consensusAlgorithm && (
                    <div className="info-item">
                      <span className="label">Consensus:</span>
                      <span className="value">{block.consensusAlgorithm}</span>
                    </div>
                  )}
                </div>
              </div>
              <div className="block-footer">
                <button className="details-button">View Details</button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {selectedBlock && (
        <div className="block-details-overlay">
          <div className="block-details-modal">
            <div className="modal-header">
              <h3>Block #{selectedBlock.height} Details</h3>
              <button className="close-button" onClick={closeBlockDetails}>
                <i className="fas fa-times"></i>
              </button>
            </div>
            <div className="modal-body">
              <div className="detail-group">
                <h4>Block Information</h4>
                <div className="detail-item">
                  <span className="label">Hash:</span>
                  <span className="value">{selectedBlock.hash}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Previous Hash:</span>
                  <span className="value">{selectedBlock.previousHash}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Timestamp:</span>
                  <span className="value">{formatDate(selectedBlock.timestamp)}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Size:</span>
                  <span className="value">{selectedBlock.size} bytes</span>
                </div>
                {selectedBlock.merkleRoot && (
                  <div className="detail-item">
                    <span className="label">Merkle Root:</span>
                    <span className="value">{selectedBlock.merkleRoot}</span>
                  </div>
                )}
                {selectedBlock.version && (
                  <div className="detail-item">
                    <span className="label">Version:</span>
                    <span className="value">{selectedBlock.version}</span>
                  </div>
                )}
              </div>
              
              <div className="detail-group">
                <h4>Consensus Information</h4>
                <div className="detail-item">
                  <span className="label">Algorithm:</span>
                  <span className="value">{selectedBlock.consensusAlgorithm || 'Hybrid'}</span>
                </div>
                {selectedBlock.validator && (
                  <div className="detail-item">
                    <span className="label">Validator:</span>
                    <span className="value">{selectedBlock.validator}</span>
                  </div>
                )}
                {selectedBlock.consistency && (
                  <div className="detail-item">
                    <span className="label">Consistency Level:</span>
                    <span className="value">{selectedBlock.consistency}</span>
                  </div>
                )}
              </div>

              <div className="detail-group">
                <h4>Transactions ({selectedBlock.transactions?.length || 0})</h4>
                {selectedBlock.transactions && selectedBlock.transactions.length > 0 ? (
                  <div className="transactions-list">
                    {selectedBlock.transactions.map((tx, index) => (
                      <div key={index} className="transaction-item">
                        <div className="tx-header">
                          <span className="tx-id">{truncateHash(tx.id)}</span>
                          <span className="tx-time">{formatDate(tx.timestamp)}</span>
                        </div>
                        <div className="tx-data">
                          <pre>{JSON.stringify(tx.data, null, 2)}</pre>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="no-transactions">No transactions in this block</p>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      <style jsx>{`
        .blocks-container {
          padding: 2rem;
          position: relative;
        }
        
        h2 {
          margin-bottom: 2rem;
          color: var(--text-color);
          font-weight: 600;
        }
        
        .blocks-timeline {
          display: flex;
          flex-direction: column;
          gap: 1.5rem;
        }
        
        .block-item {
          display: flex;
          align-items: flex-start;
          cursor: pointer;
        }
        
        .block-icon {
          background-color: var(--primary-color);
          color: white;
          width: 40px;
          height: 40px;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          margin-right: 1.5rem;
          z-index: 1;
          box-shadow: 0 0 0 4px rgba(52, 152, 219, 0.2);
        }
        
        .block-connector {
          position: absolute;
          left: 2rem + 20px;
          width: 2px;
          height: calc(100% - 40px);
          z-index: 0;
        }
        
        .connector-line {
          width: 2px;
          background-color: var(--border-color);
          height: calc(100% + 1.5rem);
          margin-left: 19px;
        }
        
        .block-card {
          flex: 1;
          background-color: var(--card-background);
          border-radius: 8px;
          box-shadow: var(--shadow);
          overflow: hidden;
          transition: transform 0.2s, box-shadow 0.2s;
        }
        
        .block-item:hover .block-card {
          transform: translateY(-2px);
          box-shadow: 0 6px 12px rgba(0, 0, 0, 0.1);
        }
        
        .block-header {
          background-color: rgba(52, 152, 219, 0.1);
          padding: 0.75rem 1rem;
          display: flex;
          justify-content: space-between;
          align-items: center;
          border-bottom: 1px solid var(--border-color);
        }
        
        .block-height {
          font-weight: 600;
          color: var(--primary-color);
        }
        
        .block-time {
          font-size: 0.85rem;
          color: var(--text-secondary);
        }
        
        .block-body {
          padding: 1rem;
        }
        
        .block-info {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
          gap: 1rem;
        }
        
        .info-item {
          display: flex;
          flex-direction: column;
        }
        
        .label {
          font-size: 0.75rem;
          color: var(--text-secondary);
          margin-bottom: 0.25rem;
        }
        
        .value {
          font-weight: 500;
        }
        
        .value.hash {
          font-family: monospace;
          font-size: 0.9rem;
          color: var(--primary-color);
        }
        
        .block-footer {
          padding: 0.75rem 1rem;
          border-top: 1px solid var(--border-color);
          display: flex;
          justify-content: flex-end;
        }
        
        .details-button {
          background-color: var(--primary-color);
          color: white;
          border: none;
          border-radius: 4px;
          padding: 0.5rem 1rem;
          font-size: 0.85rem;
          font-weight: 500;
          cursor: pointer;
          transition: background-color 0.2s;
        }
        
        .details-button:hover {
          background-color: var(--primary-dark);
        }
        
        /* Block details modal */
        .block-details-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background-color: rgba(0, 0, 0, 0.5);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 1000;
          backdrop-filter: blur(3px);
        }
        
        .block-details-modal {
          background-color: var(--card-background);
          border-radius: 8px;
          box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
          width: 90%;
          max-width: 800px;
          max-height: 90vh;
          overflow-y: auto;
          animation: modalFadeIn 0.3s ease-out;
        }
        
        @keyframes modalFadeIn {
          from {
            opacity: 0;
            transform: translateY(20px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
        
        .modal-header {
          padding: 1rem 1.5rem;
          display: flex;
          justify-content: space-between;
          align-items: center;
          border-bottom: 1px solid var(--border-color);
          position: sticky;
          top: 0;
          background-color: var(--card-background);
          z-index: 10;
        }
        
        .modal-header h3 {
          font-weight: 600;
          margin: 0;
        }
        
        .close-button {
          background: none;
          border: none;
          color: var(--text-secondary);
          font-size: 1.2rem;
          cursor: pointer;
          padding: 0.5rem;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: 50%;
          transition: background-color 0.2s;
        }
        
        .close-button:hover {
          background-color: rgba(0, 0, 0, 0.05);
          color: var(--text-color);
        }
        
        .modal-body {
          padding: 1.5rem;
        }
        
        .detail-group {
          margin-bottom: 2rem;
        }
        
        .detail-group h4 {
          margin: 0 0 1rem;
          font-weight: 600;
          color: var(--primary-color);
          padding-bottom: 0.5rem;
          border-bottom: 1px solid var(--border-color);
        }
        
        .detail-item {
          display: flex;
          margin-bottom: 0.75rem;
          flex-wrap: wrap;
        }
        
        .detail-item .label {
          width: 140px;
          min-width: 140px;
          font-weight: 500;
          color: var(--text-color);
          font-size: 0.9rem;
        }
        
        .detail-item .value {
          flex: 1;
          font-family: monospace;
          word-break: break-all;
        }
        
        .transactions-list {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }
        
        .transaction-item {
          background-color: rgba(0, 0, 0, 0.02);
          border: 1px solid var(--border-color);
          border-radius: 4px;
          overflow: hidden;
        }
        
        .tx-header {
          background-color: rgba(0, 0, 0, 0.05);
          padding: 0.75rem;
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 0.9rem;
        }
        
        .tx-id {
          font-family: monospace;
          font-weight: 500;
        }
        
        .tx-time {
          color: var(--text-secondary);
          font-size: 0.8rem;
        }
        
        .tx-data {
          padding: 0.75rem;
          max-height: 200px;
          overflow-y: auto;
        }
        
        .tx-data pre {
          margin: 0;
          font-size: 0.85rem;
          white-space: pre-wrap;
        }
        
        .no-transactions {
          color: var(--text-secondary);
          font-style: italic;
          text-align: center;
          padding: 1rem;
        }
        
        @media (max-width: 768px) {
          .blocks-container {
            padding: 1rem;
          }
          
          .block-info {
            grid-template-columns: 1fr;
          }
          
          .detail-item {
            flex-direction: column;
          }
          
          .detail-item .label {
            width: 100%;
            margin-bottom: 0.25rem;
          }
        }
      `}</style>
    </div>
  );
};

export default Blocks;