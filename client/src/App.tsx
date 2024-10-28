import '@mantine/core/styles.css';
import HomePage from '.';
import Register from '@/pages/register';
import Login from '@/pages/login';
import AboutPage from '@/pages/about';
import ContactPage from '@/pages/contact';
import Blog from '@/pages/blog';
import BlogPost from '@/pages/blog/post';
import DashboardLayout from '@/pages/dashboard/layout';
import Dashboard from '@/pages/dashboard';
import DashboardUploads from '@/pages/dashboard/uploads';
import DashboardBlog from '@/pages/dashboard/blog';
import DashboardBlogEdit from '@/pages/dashboard/blog/edit/post';
import DashboardProfile from '@/pages/dashboard/profile';
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
              <HomePage />
            }
          />
          <Route path="/about" element={<AboutPage />} />
          <Route path="/contact" element={<ContactPage />} />
          <Route path="/blog" element={<Blog />}>
              <Route path=":slug" element={<BlogPost />} />
            </Route>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/dashboard" element={<DashboardLayout />}>
            <Route index element={<Dashboard />} />
            <Route path="profile" element={<DashboardProfile />} />
            <Route path="blog" element={<DashboardBlog />}>
              <Route path="edit/:slug" element={<DashboardBlogEdit/>} />
            </Route>
            <Route path="uploads" element={<DashboardUploads />} />
          </Route>
        </Routes>
      </Layout>
    </div>
  );
}

export default App;
