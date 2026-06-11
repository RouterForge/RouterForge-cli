const userService = require('../userService');
const User = require('../../models/User');
const bcrypt = require('bcrypt');
const jwt = require('jsonwebtoken');

jest.mock('../../models/User');
jest.mock('bcrypt');
jest.mock('jsonwebtoken');

describe('User Service', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('registerUser', () => {
    it('should create user successfully and return token', async () => {
      const userData = { email: 'test@test.com', password: 'Password123!' };
      const hashedPassword = 'hashedPassword123';
      bcrypt.hash.mockResolvedValue(hashedPassword);
      const savedUser = { _id: 'userId', email: userData.email, password: hashedPassword };
      User.findOne.mockResolvedValue(null);
      User.prototype.save = jest.fn().mockResolvedValue(savedUser);
      jwt.sign.mockReturnValue('token');

      const result = await userService.registerUser(userData);

      expect(bcrypt.hash).toHaveBeenCalledWith(userData.password, 10);
      expect(User.prototype.save).toHaveBeenCalled();
      expect(jwt.sign).toHaveBeenCalledWith({ id: savedUser._id }, process.env.JWT_SECRET || 'secret', { expiresIn: '1h' });
      expect(result).toEqual({ user: savedUser, token: 'token' });
    });

    it('should throw error if user already exists', async () => {
      const userData = { email: 'existing@test.com', password: 'Password123!' };
      User.findOne.mockResolvedValue({ email: userData.email });

      await expect(userService.registerUser(userData)).rejects.toThrow('User already exists');
    });

    it('should throw error on validation failure', async () => {
      const userData = { email: 'invalid', password: 'short' };
      await expect(userService.registerUser(userData)).rejects.toThrow('Invalid email or password');
    });
  });

  describe('loginUser', () => {
    it('should return token on successful login', async () => {
      const credentials = { email: 'test@test.com', password: 'Password123!' };
      const user = { _id: 'userId', email: credentials.email, password: 'hashedPassword' };
      User.findOne.mockResolvedValue(user);
      bcrypt.compare.mockResolvedValue(true);
      jwt.sign.mockReturnValue('token');

      const result = await userService.loginUser(credentials);

      expect(bcrypt.compare).toHaveBeenCalledWith(credentials.password, user.password);
      expect(jwt.sign).toHaveBeenCalledWith({ id: user._id }, process.env.JWT_SECRET || 'secret', { expiresIn: '1h' });
      expect(result).toEqual({ user, token: 'token' });
    });

    it('should throw error if user not found', async () => {
      User.findOne.mockResolvedValue(null);
      await expect(userService.loginUser({ email: 'notfound@test.com', password: 'pass' })).rejects.toThrow('Invalid credentials');
    });

    it('should throw error if password does not match', async () => {
      User.findOne.mockResolvedValue({ email: 'test@test.com', password: 'hashedPassword' });
      bcrypt.compare.mockResolvedValue(false);
      await expect(userService.loginUser({ email: 'test@test.com', password: 'wrong' })).rejects.toThrow('Invalid credentials');
    });
  });

  describe('resetPassword', () => {
    it('should change password successfully', async () => {
      const userId = 'userId';
      const newPassword = 'NewPass123!';
      const hashedPassword = 'hashedNewPass';
      bcrypt.hash.mockResolvedValue(hashedPassword);
      User.findByIdAndUpdate.mockResolvedValue({ _id: userId, password: hashedPassword });

      const result = await userService.resetPassword(userId, newPassword);

      expect(bcrypt.hash).toHaveBeenCalledWith(newPassword, 10);
      expect(User.findByIdAndUpdate).toHaveBeenCalledWith(userId, { password: hashedPassword });
      expect(result).toEqual({ message: 'Password reset successful' });
    });

    it('should throw error if user not found', async () => {
      User.findByIdAndUpdate.mockResolvedValue(null);
      await expect(userService.resetPassword('nonexistent', 'pass')).rejects.toThrow('User not found');
    });
  });
});