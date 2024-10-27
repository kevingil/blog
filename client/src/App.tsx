import React from 'react';
import './App.css';
import { Login } from './features/auth/login';
import { Route, Routes } from 'react-router-dom';
import { Counter } from './features/counter/Counter';
import { Register } from './features/auth/register';

function App() {
  return (
    <div className="App">
      <Routes>
        <Route
          path="/"
          element={
              <Counter />
          }
        />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
      </Routes>
    </div>
  );
}

export default App;
