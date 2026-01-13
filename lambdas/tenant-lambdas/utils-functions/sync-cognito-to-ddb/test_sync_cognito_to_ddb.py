"""
Unit tests for Cognito to DynamoDB sync Lambda function
"""

import json
import os
import unittest
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime
from botocore.exceptions import ClientError

# Set environment variable before importing the module
os.environ['EMPLOYEE_TABLE'] = 'TestEmployeeTable'

from sync_cognito_to_ddb import (
    handler,
    extract_user_data,
    create_user_in_ddb,
    update_user_in_ddb,
    handle_post_confirmation,
    handle_pre_authentication,
    handle_pre_token_generation
)


class TestExtractUserData(unittest.TestCase):
    """Test user data extraction from Cognito events."""
    
    def test_extract_complete_user_data(self):
        """Test extraction with all fields present."""
        event = {
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123-456-789',
                    'email': 'test@example.com',
                    'custom:userName': 'TestUser',
                    'name': 'Test User',
                    'given_name': 'Test',
                    'family_name': 'User',
                    'custom:E_ID': 'EMP001',
                    'phone_number': '+1234567890'
                }
            }
        }
        
        result = extract_user_data(event)
        
        self.assertEqual(result['cognito_username'], 'testuser')
        self.assertEqual(result['cognito_id'], '123-456-789')
        self.assertEqual(result['email'], 'test@example.com')
        self.assertEqual(result['name'], 'Test User')
        self.assertEqual(result['given_name'], 'Test')
        self.assertEqual(result['family_name'], 'User')
        self.assertEqual(result['e_id'], 'EMP001')
        self.assertEqual(result['phone_number'], '+1234567890')
    
    def test_extract_minimal_user_data(self):
        """Test extraction with only required fields."""
        event = {
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123-456-789',
                    'email': 'test@example.com'
                }
            }
        }
        
        result = extract_user_data(event)
        
        self.assertEqual(result['cognito_username'], 'testuser')
        self.assertEqual(result['email'], 'test@example.com')
        self.assertEqual(result['name'], '')
        self.assertEqual(result['given_name'], '')
        self.assertEqual(result['family_name'], '')
        self.assertEqual(result['e_id'], '')
        self.assertEqual(result['phone_number'], '')
    
    def test_extract_builds_name_from_first_last(self):
        """Test that name is built from given_name and family_name if name is not provided."""
        event = {
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123-456-789',
                    'email': 'test@example.com',
                    'given_name': 'John',
                    'family_name': 'Doe'
                }
            }
        }
        
        result = extract_user_data(event)
        
        self.assertEqual(result['given_name'], 'John')
        self.assertEqual(result['family_name'], 'Doe')
        self.assertEqual(result['name'], 'John Doe')


class TestCreateUserInDDB(unittest.TestCase):
    """Test user creation in DynamoDB."""
    
    @patch('sync_cognito_to_ddb.table')
    def test_create_user_success(self, mock_table):
        """Test successful user creation."""
        user_data = {
            'cognito_username': 'testuser',
            'cognito_id': '123-456',
            'email': 'test@example.com',
            'name': 'Test User',
            'given_name': 'Test',
            'family_name': 'User',
            'user_name': 'testuser',
            'e_id': 'EMP001',
            'phone_number': '+1234567890'
        }
        
        mock_table.put_item = Mock()
        
        result = create_user_in_ddb(user_data)
        
        mock_table.put_item.assert_called_once()
        call_args = mock_table.put_item.call_args
        item = call_args[1]['Item']
        
        self.assertEqual(item['UserName'], 'testuser')
        self.assertEqual(item['CognitoId'], '123-456')
        self.assertEqual(item['FirstName'], 'Test')
        self.assertEqual(item['LastName'], 'User')
        self.assertEqual(item['EmailId'], 'test@example.com')
        self.assertEqual(item['Status'], 'Active')
        self.assertIn('CreatedAt', item)
    
    @patch('sync_cognito_to_ddb.table', None)
    def test_create_user_no_table(self):
        """Test creation fails when table is not configured."""
        user_data = {'cognito_username': 'test'}
        
        with self.assertRaises(ValueError):
            create_user_in_ddb(user_data)


class TestUpdateUserInDDB(unittest.TestCase):
    """Test user updates in DynamoDB."""
    
    @patch('sync_cognito_to_ddb.table')
    def test_update_existing_user(self, mock_table):
        """Test updating an existing user."""
        user_data = {
            'cognito_username': 'testuser',
            'email': 'test@example.com',
            'name': 'Updated Name',
            'given_name': 'Updated',
            'family_name': 'Name',
            'phone_number': '+9876543210',
            'e_id': 'EMP002'
        }
        
        # Mock existing user
        mock_table.get_item.return_value = {
            'Item': {'UserName': 'testuser'}
        }
        mock_table.update_item = Mock()
        
        result = update_user_in_ddb(user_data)
        
        self.assertTrue(result)
        mock_table.get_item.assert_called_once()
        mock_table.update_item.assert_called_once()
    
    @patch('sync_cognito_to_ddb.table')
    @patch('sync_cognito_to_ddb.create_user_in_ddb')
    def test_update_creates_missing_user(self, mock_create, mock_table):
        """Test that update creates user if not exists."""
        user_data = {
            'cognito_username': 'newuser',
            'email': 'new@example.com',
            'name': 'New User',
            'given_name': 'New',
            'family_name': 'User',
            'phone_number': '',
            'e_id': ''
        }
        
        # Mock user not found        # Mock user not found
        mock_table.get_item.return_value = {}
        
        result = update_user_in_ddb(user_data)
        
        self.assertTrue(result)
        mock_create.assert_called_once()


class TestCognitoTriggerHandlers(unittest.TestCase):
    """Test Cognito trigger-specific handlers."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.sample_event = {
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123-456',
                    'email': 'test@example.com',
                    'name': 'Test User'
                }
            }
        }
    
    @patch('sync_cognito_to_ddb.create_user_in_ddb')
    def test_handle_post_confirmation(self, mock_create):
        """Test PostConfirmation trigger handler."""
        handle_post_confirmation(self.sample_event)
        mock_create.assert_called_once()
    
    @patch('sync_cognito_to_ddb.update_user_in_ddb')
    def test_handle_pre_authentication(self, mock_update):
        """Test PreAuthentication trigger handler."""
        handle_pre_authentication(self.sample_event)
        mock_update.assert_called_once()
    
    @patch('sync_cognito_to_ddb.update_user_in_ddb')
    def test_handle_pre_token_generation(self, mock_update):
        """Test PreTokenGeneration trigger handler."""
        handle_pre_token_generation(self.sample_event)
        mock_update.assert_called_once()


class TestMainHandler(unittest.TestCase):
    """Test main Lambda handler function."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.context = Mock()
    
    @patch('sync_cognito_to_ddb.handle_post_confirmation')
    def test_handler_post_confirmation(self, mock_handler):
        """Test handler routes PostConfirmation correctly."""
        event = {
            'triggerSource': 'PostConfirmation_ConfirmSignUp',
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123',
                    'email': 'test@example.com'
                }
            }
        }
        
        result = handler(event, self.context)
        
        mock_handler.assert_called_once_with(event)
        self.assertEqual(result, event)
    
    @patch('sync_cognito_to_ddb.handle_pre_authentication')
    def test_handler_pre_authentication(self, mock_handler):
        """Test handler routes PreAuthentication correctly."""
        event = {
            'triggerSource': 'PreAuthentication_Authentication',
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123',
                    'email': 'test@example.com'
                }
            }
        }
        
        result = handler(event, self.context)
        
        mock_handler.assert_called_once_with(event)
        self.assertEqual(result, event)
    
    @patch('sync_cognito_to_ddb.handle_pre_token_generation')
    def test_handler_token_generation(self, mock_handler):
        """Test handler routes TokenGeneration correctly."""
        event = {
            'triggerSource': 'TokenGeneration_Authentication',
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123',
                    'email': 'test@example.com'
                }
            }
        }
        
        result = handler(event, self.context)
        
        mock_handler.assert_called_once_with(event)
        self.assertEqual(result, event)
    
    @patch('sync_cognito_to_ddb.handle_pre_token_generation')
    def test_handler_token_refresh(self, mock_handler):
        """Test handler routes TokenGeneration_RefreshTokens correctly."""
        event = {
            'triggerSource': 'TokenGeneration_RefreshTokens',
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123',
                    'email': 'test@example.com'
                }
            }
        }
        
        result = handler(event, self.context)
        
        mock_handler.assert_called_once_with(event)
        self.assertEqual(result, event)
    
    def test_handler_error_does_not_block(self):
        """Test that handler errors don't block authentication."""
        event = {
            'triggerSource': 'PostConfirmation_ConfirmSignUp',
            'userName': 'testuser',
            'request': {
                'userAttributes': {
                    'sub': '123',
                    'email': 'test@example.com'
                }
            }
        }
        
        with patch('sync_cognito_to_ddb.handle_post_confirmation', side_effect=Exception('Test error')):
            result = handler(event, self.context)
            # Should still return event despite error
            self.assertEqual(result, event)


class TestIntegration(unittest.TestCase):
    """Integration tests simulating real Cognito events."""
    
    @patch('sync_cognito_to_ddb.table')
    def test_full_user_lifecycle(self, mock_table):
        """Test complete user lifecycle: signup -> login -> token refresh."""
        context = Mock()
        
        # 1. PostConfirmation - User signs up
        signup_event = {
            'triggerSource': 'PostConfirmation_ConfirmSignUp',
            'userName': 'newuser@example.com',
            'request': {
                'userAttributes': {
                    'sub': 'abc-123',
                    'email': 'newuser@example.com',
                    'name': 'New User'
                }
            }
        }
        
        mock_table.put_item = Mock()
        result = handler(signup_event, context)
        self.assertEqual(result['userName'], 'newuser@example.com')
        mock_table.put_item.assert_called_once()
        
        # 2. PreAuthentication - User logs in
        login_event = {
            'triggerSource': 'PreAuthentication_Authentication',
            'userName': 'newuser@example.com',
            'request': {
                'userAttributes': {
                    'sub': 'abc-123',
                    'email': 'newuser@example.com',
                    'name': 'New User Updated'
                }
            }
        }
        
        mock_table.get_item.return_value = {'Item': {'UserName': 'newuser@example.com'}}
        mock_table.update_item = Mock()
        result = handler(login_event, context)
        self.assertEqual(result['userName'], 'newuser@example.com')
        mock_table.update_item.assert_called()


if __name__ == '__main__':
    unittest.main()
