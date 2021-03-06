# Pass
resource "aws_iam_policy" "policy_statement_allow_resource_without_wildcard" {
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "rds:AddRoleToDBCluster",
      "Effect": "Allow",
      "Resource": "arn:aws:rds:us-east-1:1234567890:cluster:foo_cluster"
    }
  ]
}
EOF
}

# Fail
resource "aws_iam_policy" "policy_statement_allow_resource_with_wildcard" {
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "rds:AddRoleToDBCluster",
      "Effect": "Allow",
      "Resource": "arn:aws:rds:*:*:cluster:*"
    }
  ]
}

EOF
}

# Pass
resource "aws_iam_policy" "policy_statement_deny_resource_with_wildcard" {
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "rds:AddRoleToDBCluster",
      "Effect": "Deny",
      "Resource": "arn:aws:rds:*:*:cluster:*"
    }
  ]
}

EOF
}
