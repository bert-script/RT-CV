import React from 'react'
import Document, { Html, Head, Main, NextScript } from 'next/document'
import { ServerStyleSheets } from '@material-ui/core/styles'

export default class MyDocument extends Document {
    render() {
        return (
            <Html lang="en">
                <Head>
                    <link rel="icon" href="/favicon.ico" />
                    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto:300,400,500,700&display=swap" />
                    <link rel="stylesheet" href="https://fonts.googleapis.com/icon?family=Material+Icons" />
                </Head>
                <body>
                    <Main />
                    <NextScript />
                </body>
            </Html>
        );
    }
}

// Copied over from https://github.com/mui-org/material-ui/blob/v4.x/examples/nextjs/pages/_document.js
MyDocument.getInitialProps = async (ctx) => {
    const sheets = new ServerStyleSheets();
    const originalRenderPage = ctx.renderPage;

    ctx.renderPage = () => originalRenderPage({
        enhanceApp: App => props => sheets.collect(<App {...props} />),
    });

    const initialProps = await Document.getInitialProps(ctx);

    return {
        ...initialProps,
        styles: [
            ...React.Children.toArray(initialProps.styles),
            sheets.getStyleElement(),
        ],
    };
};
