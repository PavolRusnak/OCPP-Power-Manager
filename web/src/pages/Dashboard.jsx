import React from 'react';
import { Navigate } from 'react-router-dom';

function Dashboard() {
  // Redirect to stations page since that's what we want to show on the dashboard
  return <Navigate to="/stations" replace />;
}

export default Dashboard;