import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { loginAsync, selectAuth } from '@/features/auth/authSlice';

export default function Login() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const { error } = useAppSelector(selectAuth);

  const handleSubmit = async (
    event: React.FormEvent<HTMLFormElement>
  ): Promise<void> => {
    event.preventDefault();
    const email = event.currentTarget.email.value;
    const password = event.currentTarget.password.value;

    setLoading(true);

    dispatch(loginAsync({ email, password }))
      .unwrap()
      .then(() => {
        setLoading(true);
        navigate('/');
        window.location.reload();
      })
      .catch(() => {
        setLoading(false);
      });
  };

  return (
    <>
      <main className="body">
        <div className="loginBox">
          <div className="title">Login</div>
          <div className="error">{error}</div>
          <form onSubmit={handleSubmit}>
            <input
              className="loginInput"
              type="text"
              name="email"
              placeholder="email"
              required
            ></input>
            <input
              className="loginInput"
              type="password"
              name="password"
              placeholder="Password"
              required
            ></input>
            <button className="loginButton">Log in</button>
            <div className="linkText">
              Don’t have an account yet?{' '}
              <Link className="link" to="/register">
                Sign up
              </Link>
            </div>
          </form>
        </div>
      </main>
    </>
  );
}
