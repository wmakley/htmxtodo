import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as cognito from 'aws-cdk-lib/aws-cognito';

export interface UserPoolClientStackProps extends cdk.StackProps {
	userPool: cognito.IUserPool
}

export class UserPoolClientStack extends cdk.Stack {
	constructor(scope: Construct, id: string, props: UserPoolClientStackProps) {
		super(scope, id, props);

		new cognito.UserPoolDomain(this, "UserPoolDomain", {
			userPool: props.userPool,
			cognitoDomain: {
				domainPrefix: "htmxtodo"
			}
		})

		// Create a client for use server-side:
		const client = new cognito.UserPoolClient(this, "ServerClient", {
			userPool: props.userPool,
			authFlows: {
				userSrp: true // ???
			},
		});

		// TODO: Create a client for external login flow:


		new cdk.CfnOutput(this, "UserPoolClientIdOutput", {
			value: client.userPoolClientId,
			exportName: "HtmxtodoUserPoolClientId",
		});
	}
}
