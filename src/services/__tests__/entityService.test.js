const entityService = require('../entityService');
const Entity = require('../../models/Entity');

jest.mock('../../models/Entity');

describe('Entity Service', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('createEntity', () => {
    it('should create entity successfully', async () => {
      const entityData = { name: 'Entity1', description: 'Valid description' };
      const savedEntity = { _id: 'entityId', ...entityData };
      Entity.prototype.save = jest.fn().mockResolvedValue(savedEntity);

      const result = await entityService.createEntity(entityData);

      expect(Entity.prototype.save).toHaveBeenCalled();
      expect(result).toEqual(savedEntity);
    });

    it('should throw validation error for missing name', async () => {
      const entityData = { description: 'No name' };
      await expect(entityService.createEntity(entityData)).rejects.toThrow('Name is required');
    });

    it('should throw validation error for invalid description', async () => {
      const entityData = { name: 'Valid name', description: '' };
      await expect(entityService.createEntity(entityData)).rejects.toThrow('Description is required');
    });
  });

  describe('getEntity', () => {
    it('should return entity by id', async () => {
      const entityId = 'entityId';
      const foundEntity = { _id: entityId, name: 'Entity1' };
      Entity.findById.mockResolvedValue(foundEntity);

      const result = await entityService.getEntity(entityId);

      expect(Entity.findById).toHaveBeenCalledWith(entityId);
      expect(result).toEqual(foundEntity);
    });

    it('should throw error if entity not found', async () => {
      Entity.findById.mockResolvedValue(null);
      await expect(entityService.getEntity('nonexistent')).rejects.toThrow('Entity not found');
    });

    it('should throw error for invalid id format', async () => {
      Entity.findById.mockImplementation(() => { throw new Error('Invalid ID format'); });
      await expect(entityService.getEntity('badid')).rejects.toThrow('Invalid ID format');
    });
  });

  describe('updateEntity', () => {
    it('should update entity successfully', async () => {
      const entityId = 'entityId';
      const updateData = { name: 'Updated Name' };
      const updatedEntity = { _id: entityId, ...updateData, description: 'Desc' };
      Entity.findByIdAndUpdate.mockResolvedValue(updatedEntity);

      const result = await entityService.updateEntity(entityId, updateData);

      expect(Entity.findByIdAndUpdate).toHaveBeenCalledWith(entityId, updateData, { new: true, runValidators: true });
      expect(result).toEqual(updatedEntity);
    });

    it('should throw error if entity not found', async () => {
      Entity.findByIdAndUpdate.mockResolvedValue(null);
      await expect(entityService.updateEntity('nonexistent', { name: 'New' })).rejects.toThrow('Entity not found');
    });

    it('should throw validation error for empty name', async () => {
      Entity.findByIdAndUpdate.mockImplementation(() => { throw new Error('Validation failed: name cannot be empty'); });
      await expect(entityService.updateEntity('entityId', { name: '' })).rejects.toThrow('name cannot be empty');
    });
  });

  describe('deleteEntity', () => {
    it('should delete entity successfully', async () => {
      const entityId = 'entityId';
      Entity.findByIdAndDelete.mockResolvedValue({ _id: entityId });

      const result = await entityService.deleteEntity(entityId);

      expect(Entity.findByIdAndDelete).toHaveBeenCalledWith(entityId);
      expect(result).toEqual({ message: 'Entity deleted successfully' });
    });

    it('should throw error if entity not found', async () => {
      Entity.findByIdAndDelete.mockResolvedValue(null);
      await expect(entityService.deleteEntity('nonexistent')).rejects.toThrow('Entity not found');
    });

    it('should throw error on database failure', async () => {
      Entity.findByIdAndDelete.mockRejectedValue(new Error('Database error'));
      await expect(entityService.deleteEntity('entityId')).rejects.toThrow('Database error');
    });
  });
});