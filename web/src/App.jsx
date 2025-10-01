import React, { useEffect } from 'react';
import {
  Routes,
  Route,
  useLocation
} from 'react-router-dom';

import './css/style.css';

import './charts/ChartjsConfig';

// Import pages
import Dashboard from './pages/Dashboard';
import OCPPSettings from './pages/OCPPSettings';
import Stations from './pages/Stations';
import Logs from './pages/Logs';

function App() {

  const location = useLocation();

  useEffect(() => {
    document.querySelector('html').style.scrollBehavior = 'auto'
    window.scroll({ top: 0 })
    document.querySelector('html').style.scrollBehavior = ''
  }, [location.pathname]); // triggered on route change

  return (
    <>
      <Routes>
        <Route exact path="/" element={<Dashboard />} />
        <Route exact path="/stations" element={<Stations />} />
        <Route exact path="/ocpp-settings" element={<OCPPSettings />} />
        <Route exact path="/logs" element={<Logs />} />
      </Routes>
    </>
  );
}

export default App;
