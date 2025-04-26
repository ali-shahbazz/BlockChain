import React from 'react';

const Header = () => {
  return (
    <header className="header">
      <div className="container">
        <div className="logo">
          <i className="fas fa-cubes"></i>
          <h1>Advanced Blockchain Explorer</h1>
        </div>
        <nav>
          <ul>
            <li><a href="#home" className="active"><i className="fas fa-home"></i> Dashboard</a></li>
            <li><a href="#blocks"><i className="fas fa-cube"></i> Blocks</a></li>
            <li><a href="#consensus"><i className="fas fa-network-wired"></i> Consensus</a></li>
            <li><a href="#add-transaction"><i className="fas fa-plus-circle"></i> Add Transaction</a></li>
          </ul>
        </nav>
      </div>
      <style jsx>{`
        .header {
          background-color: var(--card-background);
          box-shadow: var(--shadow);
          position: sticky;
          top: 0;
          z-index: 100;
        }
        
        .container {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 1rem 2rem;
          max-width: 1400px;
          margin: 0 auto;
        }
        
        .logo {
          display: flex;
          align-items: center;
          color: var(--primary-color);
        }
        
        .logo i {
          font-size: 2rem;
          margin-right: 1rem;
        }
        
        .logo h1 {
          font-size: 1.5rem;
          font-weight: 600;
          margin: 0;
        }
        
        nav ul {
          display: flex;
          list-style: none;
          margin: 0;
          padding: 0;
        }
        
        nav li {
          margin-left: 2rem;
        }
        
        nav a {
          color: var(--text-secondary);
          text-decoration: none;
          font-weight: 500;
          display: flex;
          align-items: center;
          transition: color 0.2s;
        }
        
        nav a i {
          margin-right: 0.5rem;
        }
        
        nav a:hover, nav a.active {
          color: var(--primary-color);
        }
        
        @media (max-width: 768px) {
          .container {
            flex-direction: column;
            padding: 1rem;
          }
          
          nav {
            margin-top: 1rem;
            width: 100%;
            overflow-x: auto;
          }
          
          nav ul {
            width: 100%;
          }
          
          nav li {
            margin-left: 1rem;
            white-space: nowrap;
          }
          
          nav li:first-child {
            margin-left: 0;
          }
        }
      `}</style>
    </header>
  );
};

export default Header;