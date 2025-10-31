# Run this script to generate base 64 encode for csv content

import base64

# Read content from a CSV file
csv_content = """UserName,EmailId,ExternalId,DisplayName,PhoneNumber,LoginType,Active,TopLevelGroupId,TopLevelGroupName,TopLevelGroupDesc,Location
john@example.com,john@example.com,12345,John Doe,1234567890,password,Yes,1,Admins,Administrators,New York
alice@example.com,alice@example.com,67890,Alice Smith,9876543210,password,Yes,2,Users,Regular Users,London
bob@example.com,bob@example.com,54321,Bob Johnson,5555555555,password,No,3,Guests,Guest Users,Paris
sarah@example.com,sarah@example.com,98765,Sarah Lee,1112223333,password,Yes,4,Managers,Management Team,Berlin
chris@example.com,chris@example.com,13579,Chris Brown,4445556666,password,Yes,5,Developers,Engineering Team,San Francisco
emily@example.com,emily@example.com,24680,Emily Davis,7778889999,password,Yes,6,Support,Support Team,Tokyo
michael@example.com,michael@example.com,36912,Michael Taylor,1010101010,password,No,7,Testers,QA Team,Seoul
laura@example.com,laura@example.com,98741,Laura Wilson,1212121212,password,Yes,8,Marketing,Marketing Team,Los Angeles
kevin@example.com,kevin@example.com,24689,Kevin Martinez,1313131313,password,Yes,9,Finance,Finance Team,Mumbai
jessica@example.com,jessica@example.com,95162,Jessica Lopez,1414141414,password,No,10,Interns,Internship Program,Shanghai"""

# Encode the CSV content to base64
encoded_content = base64.b64encode(csv_content.encode()).decode()

print(encoded_content)
