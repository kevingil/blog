import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { loginAsync, registerAsync, selectAuth } from '@/features/auth/authSlice';

export default function Register() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const { error } = useAppSelector(selectAuth);

  const handleSubmit = async (
    event: React.FormEvent<HTMLFormElement>
  ): Promise<void> => {
    event.preventDefault();
    const name = event.currentTarget.name.valueOf();
    const email = event.currentTarget.email.value;
    const password = event.currentTarget.password.value;
    const passwordConf = event.currentTarget.passwordConf.value;

    setLoading(true);

    dispatch(registerAsync({ name, email, password, passwordConf }))
      .unwrap()
      .then(() => {
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
      })
      .catch(() => {
        setLoading(false);
      });
  };

  return (
    <>
      <main className="body">
        <div className="loginBox">
          <div className="title">Register</div>
          <div className="error">{error}</div>
          <form onSubmit={handleSubmit}>
            <input
              className="loginInput"
              type="name"
              name="name"
              placeholder="Name"
              required
            ></input>
            <input
              className="loginInput"
              type="email"
              name="email"
              placeholder="Email"
              required
            ></input>
            <input
              className="loginInput"
              type="password"
              name="password"
              placeholder="Password"
              required
            ></input>
            <input
              className="loginInput"
              type="password"
              name="passwordConf"
              placeholder="Repeat Password"
              required
            ></input>
            <button className="loginButton">Sign up</button>
            <div className="linkText">
              Already have an account?{' '}
              <Link className="link" to="/login">
                Sign in
              </Link>
            </div>
          </form>
        </div>
      </main>
    </>
  );
}
