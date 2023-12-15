import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as cognito from 'aws-cdk-lib/aws-cognito';

export class UserPoolStack extends cdk.Stack {
	public userPool: cognito.UserPool
	constructor(scope: Construct, id: string, props?: cdk.StackProps) {
		super(scope, id, props);

		this.userPool = new cognito.UserPool(this, "UserPool", {
			accountRecovery: cognito.AccountRecovery.EMAIL_ONLY,
			signInAliases: {
				email: true,
				username: false,
			},
			userPoolName: "HtmxtodoUserPool",
			selfSignUpEnabled: true,
		});

		// new cognito.UserPoolIdentityProviderAmazon(this, "ProviderAmazon", {
		// 	userPool: this.userPool,
		// 	clientId: "TODO",
		// 	clientSecret: "TODO",
		// });
		// new cognito.UserPoolIdentityProviderApple(this, "ProviderApple", {
		// 	userPool: this.userPool,
		// 	clientId: "TODO",
		// 	teamId: "TODO",
		// 	keyId: "TODO",
		// 	privateKey: "TODO",
		// });
		// new cognito.UserPoolIdentityProviderFacebook(this, "ProviderFacebook", {
		// 	userPool: this.userPool,
		// 	clientId: "TODO",
		// 	clientSecret: "TODO",
		// });
		// new cognito.UserPoolIdentityProviderGoogle(this, "ProviderGoogle", {
		// 	userPool: this.userPool,
		// 	clientId: "TODO",
		// 	clientSecretValue: "TODO"
		// });

		// might need in future:
		new cdk.CfnOutput(this, "UserPoolArnOutput", {
			value: this.userPool.userPoolArn,
			exportName: "HtmxtodoUserPoolArn",
		});

		new cdk.CfnOutput(this, "UserPoolIdOutput", {
			value: this.userPool.userPoolId,
			exportName: "HtmxtodoUserPoolId",
		});
	}
}
