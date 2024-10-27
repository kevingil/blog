import '@mantine/core/styles.css';
import { Counter } from './features/counter/Counter';
import { Register } from '@/pages/register';
import { Login } from '@/pages/login';
import AboutPage from '@/pages/about';
import ContactPage from '@/pages/contact';
import { Blog } from '@/pages/blog';
import './App.css';
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
            <Route path="/about" element={<AboutPage />} />
            <Route path="/contact" element={<ContactPage />} />
            <Route path="/blog" element={<Blog />} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
          </Routes>
      </Layout>
    </div>
  );
}

export default App;
