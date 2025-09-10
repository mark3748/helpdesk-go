# Asset Management API - Added to OpenAPI Specification

This document summarizes the asset management API endpoints and schemas that were added to the OpenAPI specification.

## New API Endpoints Added

### Asset Categories
- `GET /asset-categories` - List asset categories
- `POST /asset-categories` - Create asset category (admin/manager)
- `GET /asset-categories/{id}` - Get asset category

### Asset Management
- `GET /assets` - List assets with filtering and pagination
- `POST /assets` - Create asset (admin/manager)
- `GET /assets/{id}` - Get asset details
- `PATCH /assets/{id}` - Update asset (admin/manager) 
- `DELETE /assets/{id}` - Delete asset (admin)
- `POST /assets/{id}/assign` - Assign asset to user (admin/manager)

### Asset Checkout/Checkin
- `POST /assets/{id}/checkout` - Checkout asset (admin/manager)
- `POST /assets/checkin` - Checkin asset (admin/manager)
- `GET /assets/checkouts/active` - List active checkouts
- `GET /assets/checkouts/overdue` - List overdue checkouts

### Asset History & Assignments
- `GET /assets/{id}/history` - Get asset history with pagination
- `GET /assets/{id}/assignments` - Get asset assignment history

## New Schemas Added

### Core Asset Types
- `AssetCondition` - Enum: excellent, good, fair, poor, broken
- `AssetStatus` - Enum: active, inactive, maintenance, retired, disposed
- `Asset` - Complete asset object with all properties
- `AssetCategory` - Asset category with hierarchical support
- `AssetUser` - User info for asset context
- `AssetListResponse` - Paginated asset list response

### Request/Response Types
- `CreateAssetRequest` - Asset creation payload
- `UpdateAssetRequest` - Asset update payload
- `AssignAssetRequest` - Asset assignment payload
- `CreateCategoryRequest` - Category creation payload

### Checkout/Checkin Types
- `AssetCheckout` - Complete checkout record
- `CheckoutRequest` - Asset checkout payload
- `CheckinRequest` - Asset checkin payload

### History & Audit Types
- `AssetHistory` - Asset history/audit record
- `AssetAssignment` - Asset assignment record

## Features Supported

### Asset Management
- ✅ Full CRUD operations for assets
- ✅ Asset categorization with hierarchical support
- ✅ Custom fields for flexible asset data
- ✅ Financial tracking (purchase price, depreciation, etc.)
- ✅ Physical details (serial number, model, manufacturer, location)
- ✅ Asset status and condition tracking

### Asset Assignment & Checkout
- ✅ Asset assignment to users
- ✅ Asset checkout/checkin workflow
- ✅ Condition tracking at checkout/checkin
- ✅ Expected return dates and overdue tracking
- ✅ Approval workflow support

### Search & Filtering
- ✅ Text search across asset properties
- ✅ Filter by category, status, condition, assigned user
- ✅ Filter by manufacturer, location
- ✅ Pagination and sorting support

### Audit & History
- ✅ Complete audit trail for all asset changes
- ✅ Assignment history tracking
- ✅ Pagination for history records

## Role-Based Access Control

- **Read Operations**: All authenticated users
- **Asset CRUD**: Admin and Manager roles
- **Asset Assignment/Checkout**: Admin and Manager roles  
- **Asset Deletion**: Admin role only

## Generated TypeScript Types

TypeScript types have been generated for both frontend applications:
- `/web/internal/src/types/openapi.ts` - Internal frontend types
- `/web/requester/src/types/openapi.ts` - Requester frontend types

These include full type safety for all asset management operations, requests, and responses.

## Usage

After regenerating types, frontend developers can now use fully typed asset management APIs:

```typescript
// Example usage in TypeScript
import type { components } from './types/openapi';

type Asset = components['schemas']['Asset'];
type CreateAssetRequest = components['schemas']['CreateAssetRequest'];
type AssetCheckout = components['schemas']['AssetCheckout'];

// Fully typed API calls are now available
```

The API endpoints follow RESTful conventions and include proper HTTP status codes, error handling, and security controls.
