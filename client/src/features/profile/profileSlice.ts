// features/profile/profileSlice.ts
import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { UserProfile, UserProfileState, UpdateProfileData } from 'ProfileModels';
import http from '@/services/http';

const initialState: UserProfileState = {
  profile: null,
  loading: false,
  error: null,
  updateSuccess: false,
};

// Fetch the current user's profile
export const fetchUserProfile = createAsyncThunk(
  'profile/fetchUserProfile',
  async (_, { rejectWithValue }) => {
    try {
      const response = await http.get<UserProfile>('/api/v1/users/me');
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to fetch profile');
    }
  }
);

// Update user profile
export const updateUserProfile = createAsyncThunk(
  'profile/updateUserProfile',
  async (profileData: UpdateProfileData, { rejectWithValue }) => {
    try {
      const response = await http.put<UserProfile>('/api/v1/users/me', profileData);
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to update profile');
    }
  }
);

// Upload avatar
export const uploadAvatar = createAsyncThunk(
  'profile/uploadAvatar',
  async (file: File, { rejectWithValue }) => {
    try {
      const formData = new FormData();
      formData.append('avatar', file);

      const response = await http.post<{ avatar_url: string }>(
        '/api/v1/users/me/avatar',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      );

      return response.data.avatar_url;
    } catch (error) {
      return rejectWithValue('Failed to upload avatar');
    }
  }
);

// Delete avatar
export const deleteAvatar = createAsyncThunk(
  'profile/deleteAvatar',
  async (_, { rejectWithValue }) => {
    try {
      await http.delete('/api/v1/users/me/avatar');
      return null;
    } catch (error) {
      return rejectWithValue('Failed to delete avatar');
    }
  }
);

const profileSlice = createSlice({
  name: 'profile',
  initialState,
  reducers: {
    clearUpdateSuccess: (state) => {
      state.updateSuccess = false;
    },
    clearProfileErrors: (state) => {
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    builder
      // Fetch Profile
      .addCase(fetchUserProfile.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchUserProfile.fulfilled, (state, action) => {
        state.loading = false;
        state.profile = action.payload;
      })
      .addCase(fetchUserProfile.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      })
      // Update Profile
      .addCase(updateUserProfile.pending, (state) => {
        state.loading = true;
        state.error = null;
        state.updateSuccess = false;
      })
      .addCase(updateUserProfile.fulfilled, (state, action) => {
        state.loading = false;
        state.profile = action.payload;
        state.updateSuccess = true;
      })
      .addCase(updateUserProfile.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
        state.updateSuccess = false;
      })
      // Upload Avatar
      .addCase(uploadAvatar.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(uploadAvatar.fulfilled, (state, action) => {
        state.loading = false;
        if (state.profile) {
          state.profile.avatar = action.payload;
        }
      })
      .addCase(uploadAvatar.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      })
      // Delete Avatar
      .addCase(deleteAvatar.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(deleteAvatar.fulfilled, (state) => {
        state.loading = false;
        if (state.profile) {
          state.profile.avatar = null;
        }
      })
      .addCase(deleteAvatar.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      });
  },
});

export const { clearUpdateSuccess, clearProfileErrors } = profileSlice.actions;
export default profileSlice.reducer;
