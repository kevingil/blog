import axios from 'axios';

const httpPublic = axios.create({
  baseURL: 'http://localhost:5001',
  headers: {
    'Content-Type': 'application/json'
  }
});

export default httpPublic;
