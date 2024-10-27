import '@mantine/core/styles.css';
import { Counter } from './features/counter/Counter';
import { Register } from './features/auth/register';
import { Login } from './features/auth/login'; import './App.css';
import Layout from './components/Layout';
import { Route, Routes } from 'react-router-dom';


function App() {
  return (
    <div className="App">
      <Layout>
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
      </Layout>
    </div>
  );
}

export default App;
