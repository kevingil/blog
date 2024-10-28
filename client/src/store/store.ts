import { configureStore, ThunkAction, Action } from '@reduxjs/toolkit';
import counterReducer from '@/features/counter/counterSlice';
import authReducer from '@/features/auth/authSlice';
import blogReducer from '@/features/blog/blogSlice';
import profileReducer from '@/features/profile/profileSlice';
import storageReducer from '@/features/storage/storageSlice';

export const store = configureStore({
  reducer: {
    counter: counterReducer,
    auth: authReducer,
    blog: blogReducer,
    profile: profileReducer,
    storage: storageReducer,
  }
});

export type AppDispatch = typeof store.dispatch;
export type RootState = ReturnType<typeof store.getState>;
export type AppThunk<ReturnType = void> = ThunkAction<
  ReturnType,
  RootState,
  unknown,
  Action<string>
>;
